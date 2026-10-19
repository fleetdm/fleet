#!/usr/bin/env python3
"""Collect a compact DEX snapshot from the Fleet REST API.

Reads FLEET_URL and FLEET_API_TOKEN from the environment. Only issues GET
requests. Writes one JSON snapshot that the dashboard builders consume.

Endpoints used (API-only user allowlist):
  GET /api/v1/fleet/hosts            List hosts (populate_software)
  GET /api/v1/fleet/hosts/:id        Get host (exclude_software)
  GET /api/v1/fleet/fleets           List fleets
  GET /api/v1/fleet/reports          List reports
  GET /api/v1/fleet/reports/:id/report  Get report data
"""
import argparse
import concurrent.futures
import gzip
import datetime as dt
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

DEVICE_HEALTH_REPORT = "DEX - Hardware experience - Device health"
OS_HEALTH_REPORT = "DEX - System experience - OS health"
AI_TOOLS_REPORT = "DEX - AI governance - AI tools per host"
# An earlier report that returns one row per tool. Fleet keeps at most 1,000
# rows per report, so it only covers part of a larger fleet.
AI_TOOLS_ROWS_REPORT = "DEX - AI Tools - Surfaces AI tools on the host"
# One row per host each. Connectivity and Analytics read these.
DEX_REPORTS = {
    "wifi": "DEX - Network experience - Wi-Fi signal quality",
    "vpn": "DEX - Network experience - VPN gate",
    "mac_posture": "DEX - System experience - Security posture",
    "win_security": "DEX - Security compliance - Windows firewall status",
    "win_secure_boot": "DEX - Security compliance - Secure Boot status (Windows)",
    "win_tpm": "DEX - Security compliance - TPM status (Windows)",
}
AI_ROWS_COLUMNS = {"tool_type": "type", "tool_name": "name", "origin": "source", "target_endpoint": "endpoint",
                   "risk_indicators": "risk_flags"}
NOT_COMPUTERS = {"ios", "ipados", "android", "chrome"}
HOST_FIELDS = [
    "id", "display_name", "computer_name", "hostname", "platform", "os_version",
    "hardware_model", "hardware_serial", "hardware_vendor", "cpu_brand", "memory",
    "team_id", "team_name", "status", "seen_time", "uptime", "last_restarted_at",
    "last_enrolled_at", "created_at", "gigs_disk_space_available",
    "percent_disk_space_available", "software_updated_at", "detail_updated_at", "policy_updated_at",
    "disk_encryption_enabled", "orbit_version", "osquery_version", "fleet_desktop_version",
]


def die(msg):
    print(f"FATAL: {msg}", file=sys.stderr)
    sys.exit(2)


class Fleet:
    def __init__(self, url, token):
        url = url.strip().rstrip("/")
        if "://" not in url:
            url = "https://" + url.lstrip("/")
        self.url = url
        self.token = token

    def get(self, path, params=None, optional=False):
        """GET a Fleet API path. With optional=True, a 402 (needs Fleet Premium)
        or 404 returns None instead of stopping the run."""
        q = ("?" + urllib.parse.urlencode(params)) if params else ""
        req = urllib.request.Request(
            self.url + path + q,
            headers={"Authorization": f"Bearer {self.token}", "Accept": "application/json",
                     "Accept-Encoding": "gzip"},
        )
        for attempt in range(4):
            try:
                with urllib.request.urlopen(req, timeout=120) as r:
                    body = r.read()
                    if r.headers.get("Content-Encoding") == "gzip":
                        body = gzip.decompress(body)
                    return json.loads(body)
            except urllib.error.HTTPError as e:
                body = e.read()[:300].decode("utf-8", "replace")
                if optional and e.code in (402, 404):
                    return None
                if e.code in (401, 403):
                    die(f"GET {path} returned {e.code}: {body}")
                if e.code >= 500 and attempt < 3:
                    time.sleep(2 ** attempt * 3)
                    continue
                die(f"GET {path} returned {e.code}: {body}")
            except (urllib.error.URLError, TimeoutError) as e:
                if attempt < 3:
                    time.sleep(2 ** attempt * 3)
                    continue
                die(f"GET {path} failed: {e}")


def compact_software(sw):
    out = []
    for s in sw or []:
        paths = s.get("installed_paths") or []
        out.append([
            s.get("name") or "",
            s.get("version") or "",
            s.get("source") or "",
            s.get("bundle_identifier") or "",
            s.get("last_opened_at") if s.get("last_opened_at") not in (None,) else None,
            paths[0] if paths else "",
            s.get("vendor") or "",
        ])
    return out


def report_rows(fleet, fleets, name, info=None):
    """Latest row per host from every copy of a report: global and per fleet.

    Pass a dict as info to learn whether the report exists and whether Fleet clipped it.
    """
    rows, seen = {}, set()
    info = {} if info is None else info
    info.update(found=False, clipped=False)
    scopes = [{}] + [{"fleet_id": f["id"]} for f in fleets]
    for scope in scopes:
        d = fleet.get("/api/v1/fleet/reports", {**scope, "per_page": 500, "query": name}) or {}
        for r in d.get("reports") or d.get("queries") or []:
            if r.get("name") != name or r["id"] in seen:
                continue
            seen.add(r["id"])
            info["found"] = True
            rep = fleet.get(f"/api/v1/fleet/reports/{r['id']}/report")
            info["clipped"] = info["clipped"] or bool(rep.get("report_clipped"))
            for row in rep.get("results") or []:
                hid = row.get("host_id")
                prev = rows.get(hid)
                if prev is None or (row.get("last_fetched") or "") > (prev.get("last_fetched") or ""):
                    rows[hid] = {"last_fetched": row.get("last_fetched"), **(row.get("columns") or {})}
    return rows


def ai_tools(fleet, fleets):
    """Each host's ai_tools rows, from the one-row-per-host report where it has a row."""
    hosts = {}
    for hid, r in report_rows(fleet, fleets, AI_TOOLS_REPORT).items():
        try:
            tools = json.loads(r.get("tools") or "[]")
        except ValueError:
            tools = []
        hosts[str(hid)] = {"last_fetched": r.get("last_fetched"), "tools": tools}
    per_host = set(hosts)

    # The per-tool report fills in hosts the per-host report hasn't heard from,
    # such as while a newly added per-host report is still collecting.
    clipped, seen = False, set()
    for scope in [{}] + [{"fleet_id": f["id"]} for f in fleets]:
        d = fleet.get("/api/v1/fleet/reports", {**scope, "per_page": 500, "query": AI_TOOLS_ROWS_REPORT}) or {}
        for r in d.get("reports") or d.get("queries") or []:
            if r.get("name") != AI_TOOLS_ROWS_REPORT or r["id"] in seen:
                continue
            seen.add(r["id"])
            rep = fleet.get(f"/api/v1/fleet/reports/{r['id']}/report")
            clipped = clipped or bool(rep.get("report_clipped"))
            for row in rep.get("results") or []:
                hid = str(row.get("host_id"))
                if hid in per_host:
                    continue
                h = hosts.setdefault(hid, {"last_fetched": row.get("last_fetched"), "tools": []})
                h["last_fetched"] = max(h["last_fetched"] or "", row.get("last_fetched") or "")
                h["tools"].append({AI_ROWS_COLUMNS.get(k, k): v for k, v in (row.get("columns") or {}).items()})
    report = AI_TOOLS_REPORT if per_host else AI_TOOLS_ROWS_REPORT if hosts else None
    return {"report": report, "clipped": clipped and not per_host, "per_host": len(per_host), "hosts": hosts}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--per-page", type=int, default=20)
    args = ap.parse_args()

    url, token = os.environ.get("FLEET_URL"), os.environ.get("FLEET_API_TOKEN")
    if not url or not token:
        die("FLEET_URL and FLEET_API_TOKEN must be set")
    fleet = Fleet(url, token)

    # Fleet Free has no fleets; List fleets answers 402 there.
    fl = fleet.get("/api/v1/fleet/fleets", {"per_page": 500}, optional=True) or {}
    fleets = fl.get("fleets") or fl.get("teams") or []

    hosts, software = [], {}
    page = 0
    while True:
        d = fleet.get("/api/v1/fleet/hosts", {
            "per_page": args.per_page, "page": page, "order_key": "id",
            "populate_software": "without_vulnerability_details",
        })
        batch = d.get("hosts") or []
        for h in batch:
            row = {k: h.get(k) for k in HOST_FIELDS}
            row["failing_policies"] = (h.get("issues") or {}).get("failing_policies_count")
            row["mdm_status"] = (h.get("mdm") or {}).get("enrollment_status")
            hosts.append(row)
            software[str(h["id"])] = compact_software(h.get("software"))
        if len(batch) < args.per_page:
            break
        page += 1

    # Get host adds custom host vitals (e.g. warranty dates) and battery health.
    def detail(h):
        g = fleet.get(f"/api/v1/fleet/hosts/{h['id']}", {"exclude_software": "true"}).get("host") or {}
        return str(h["id"]), {
            "vitals": {v.get("name"): v.get("value") for v in g.get("custom_host_vitals") or []},
            "batteries": g.get("batteries") or [],
            "hardware_marketing_name": g.get("hardware_marketing_name") or "",
            "os_update_minimum_version": g.get("os_update_minimum_version") or "",
            "os_update_deadline": g.get("os_update_deadline") or "",
        }

    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        details = dict(pool.map(detail, [h for h in hosts if h["platform"] and h["platform"] not in NOT_COMPUTERS]))

    report_info = {}

    def rows(key, name):
        report_info[key] = {}
        return report_rows(fleet, fleets, name, report_info[key])

    snap = {
        "schema": 1,
        "generated_at": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "fleet_url": fleet.url,
        "fleets": [{"id": f["id"], "name": f["name"]} for f in fleets],
        "hosts": hosts,
        "software_columns": ["name", "version", "source", "bundle_identifier", "last_opened_at", "path", "vendor"],
        "software": software,
        "host_details": details,
        "device_health": rows("device_health", DEVICE_HEALTH_REPORT),
        "os_health": rows("os_health", OS_HEALTH_REPORT),
        "ai_tools": ai_tools(fleet, fleets),
        "dex": {key: rows(key, name) for key, name in DEX_REPORTS.items()},
        "report_info": report_info,
    }
    with open(args.out, "w") as f:
        json.dump(snap, f, separators=(",", ":"))
    n_sw = sum(len(v) for v in software.values())
    print(f"hosts={len(hosts)} software_rows={n_sw} device_health={len(snap['device_health'])} "
          f"os_health={len(snap['os_health'])} ai_tools={len(snap['ai_tools']['hosts'])} "
          f"dex_reports={sum(1 for v in snap['report_info'].values() if v['found'])} "
          f"bytes={os.path.getsize(args.out)}")


if __name__ == "__main__":
    main()
