#!/usr/bin/env python3
"""Collect a compact DEX snapshot from the Fleet REST API.

Reads FLEET_URL and FLEET_API_TOKEN from the environment. Only issues GET
requests. run.py calls collect() directly; run this file on its own to write
a snapshot for debugging:

  collect.py --out snapshot.json

Endpoints used (API-only user allowlist):
  GET /api/v1/fleet/hosts            List hosts (populate_software)
  GET /api/v1/fleet/hosts/:id        Get host (exclude_software)
  GET /api/v1/fleet/fleets           List fleets
  GET /api/v1/fleet/reports          List reports
  GET /api/v1/fleet/reports/:id/report  Get report data

DEX_WORKERS (default 8) sets how many requests run at once: host details use
all of them, host pages and reports half each.
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
# An earlier report that returns one row per tool. Before Fleet 4.93.0 a report
# keeps at most 1,000 rows, so it only covers part of a larger fleet there.
AI_TOOLS_ROWS_REPORT = "DEX - AI Tools - Surfaces AI tools on the host"
# One row per host each. Connectivity and Analytics read these.
DEX_REPORTS = {
    "wifi": "DEX - Network experience - Wi-Fi signal quality",
    "vpn": "DEX - Network experience - VPN gate",
    "mac_posture": "DEX - System experience - Security posture",
    "win_security": "DEX - Security compliance - Windows firewall status",
    "win_secure_boot": "DEX - Security compliance - Secure Boot status (Windows)",
    "win_tpm": "DEX - Security compliance - TPM status (Windows)",
    "mac_energy": "DEX - Sustainability - Mac energy",
    "mac_sleep": "DEX - Sustainability - Mac sleep settings",
    "win_power": "DEX - Sustainability - Windows power",
    "lin_power": "DEX - Sustainability - Linux power",
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
SOFTWARE_COLUMNS = ["name", "version", "source", "bundle_identifier", "last_opened_at", "path", "vendor"]


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


def compact_software(sw, keep=None):
    """Software rows as lists in SOFTWARE_COLUMNS order. keep(source, name, path, bundle_id)
    drops rows no page reads, which is most of each host's inventory."""
    out = []
    for s in sw or []:
        paths = s.get("installed_paths") or []
        path = paths[0] if paths else ""
        if keep and not keep(s.get("source") or "", s.get("name") or "", path, s.get("bundle_identifier") or ""):
            continue
        out.append([
            s.get("name") or "",
            s.get("version") or "",
            s.get("source") or "",
            s.get("bundle_identifier") or "",
            s.get("last_opened_at"),
            path,
            s.get("vendor") or "",
        ])
    return out


def report_rows(fleet, fleets, name, info=None):
    """Latest row per host, keyed by host ID as a string, from every copy of a report: global and per fleet.

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
                hid = str(row.get("host_id"))
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
        hosts[hid] = {"last_fetched": r.get("last_fetched"), "tools": tools}
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


def detail(fleet, h):
    """Get host adds custom host vitals (e.g. warranty dates), battery health,
    each policy's result, and the country from GeoIP, when the Fleet server has
    a GeoIP database. The policies themselves come back under "_policy_info"."""
    g = fleet.get(f"/api/v1/fleet/hosts/{h['id']}", {"exclude_software": "true"}).get("host") or {}
    policies = [p for p in g.get("policies") or [] if p.get("id") is not None]
    return {
        # [policy ID, 1 pass / 0 fail / None no result yet]
        "policies": [[p["id"], {"pass": 1, "fail": 0}.get(p.get("response"))] for p in policies],
        "_policy_info": {str(p["id"]): {
            "name": p.get("name") or "", "description": p.get("description") or "", "resolution": p.get("resolution") or "",
            "critical": bool(p.get("critical")), "platform": p.get("platform") or "",
            "fleet_id": p.get("fleet_id", p.get("team_id"))} for p in policies},
        "vitals": {v.get("name"): v.get("value") for v in g.get("custom_host_vitals") or []},
        "batteries": g.get("batteries") or [],
        "country": (g.get("geolocation") or {}).get("country_iso") or "",
        "hardware_marketing_name": g.get("hardware_marketing_name") or "",
        "os_update_minimum_version": g.get("os_update_minimum_version") or "",
        "os_update_deadline": g.get("os_update_deadline") or "",
    }


def submit_reports(fleet, fleets, pool):
    """Start fetching every DEX report side by side. finish_reports() collects them."""
    names = {"device_health": DEVICE_HEALTH_REPORT, "os_health": OS_HEALTH_REPORT, **DEX_REPORTS}
    info = {key: {} for key in names}
    futures = {key: pool.submit(report_rows, fleet, fleets, name, info[key]) for key, name in names.items()}
    futures["ai_tools"] = pool.submit(ai_tools, fleet, fleets)
    return futures, info


def finish_reports(futures, info):
    rows = {key: f.result() for key, f in futures.items()}
    return {"device_health": rows.pop("device_health"), "os_health": rows.pop("os_health"),
            "ai_tools": rows.pop("ai_tools"), "dex": rows, "report_info": info}


def write_json(path, obj):
    tmp = path + ".tmp"
    with open(tmp, "w") as f:
        json.dump(obj, f, separators=(",", ":"))
    os.replace(tmp, path)


def collect(fleet, keep=None, workers=None, per_page=20, checkpoint=None, deadline=None):
    """Return the snapshot, or None when the deadline (a time.monotonic() value) passes first.

    With a checkpoint directory, each finished page of hosts (with their software
    and details) and the reports are saved there, so the next call skips them.
    """
    workers = workers or int(os.environ.get("DEX_WORKERS") or 8)
    if checkpoint:
        os.makedirs(os.path.join(checkpoint, "pages"), exist_ok=True)
    started = dt.datetime.now(dt.timezone.utc)

    # Fleet Free has no fleets; List fleets answers 402 there.
    fl = fleet.get("/api/v1/fleet/fleets", {"per_page": 500}, optional=True) or {}
    fleets = fl.get("fleets") or fl.get("teams") or []

    done, last = {}, None
    if checkpoint:
        for name in os.listdir(os.path.join(checkpoint, "pages")):
            if name.endswith(".json"):
                with open(os.path.join(checkpoint, "pages", name)) as f:
                    page = json.load(f)
                done[page["page"]] = page
                if page["last"]:
                    last = page["page"] if last is None else min(last, page["page"])

    def fetch_page(n):
        d = fleet.get("/api/v1/fleet/hosts", {
            "per_page": per_page, "page": n, "order_key": "id",
            "populate_software": "without_vulnerability_details",
        })
        hosts, software = [], {}
        for h in d.get("hosts") or []:
            row = {k: h.get(k) for k in HOST_FIELDS}
            row["failing_policies"] = (h.get("issues") or {}).get("failing_policies_count")
            row["mdm_status"] = (h.get("mdm") or {}).get("enrollment_status")
            hosts.append(row)
            software[str(h["id"])] = compact_software(h.get("software"), keep)
        return hosts, software

    report_pool = concurrent.futures.ThreadPoolExecutor(max_workers=max(1, workers // 2))
    report_file = os.path.join(checkpoint, "reports.json") if checkpoint else None
    reports = None
    if report_file and os.path.exists(report_file):
        with open(report_file) as f:
            reports = json.load(f)
    report_futures, report_info = submit_reports(fleet, fleets, report_pool) if reports is None else ({}, {})

    page_pool = concurrent.futures.ThreadPoolExecutor(max_workers=max(1, workers // 2))
    detail_pool = concurrent.futures.ThreadPoolExecutor(max_workers=workers)
    pending_pages, pending_details = {}, {}
    next_page = 0
    try:
        while True:
            out_of_time = deadline is not None and time.monotonic() > deadline
            while (not out_of_time and len(pending_pages) < max(1, workers // 2)
                   and (last is None or next_page <= last)):
                if next_page not in done:
                    pending_pages[next_page] = page_pool.submit(fetch_page, next_page)
                next_page += 1
            if not pending_pages and not pending_details:
                break
            waiting = list(pending_pages.values()) + [f for fs in pending_details.values() for f in fs[2].values()]
            concurrent.futures.wait(waiting, timeout=5, return_when=concurrent.futures.FIRST_COMPLETED)
            for n, f in list(pending_pages.items()):
                if not f.done():
                    continue
                del pending_pages[n]
                hosts, software = f.result()
                if len(hosts) < per_page:
                    last = n if last is None else min(last, n)
                computers = [h for h in hosts if h["platform"] and h["platform"] not in NOT_COMPUTERS]
                pending_details[n] = (hosts, software, {str(h["id"]): detail_pool.submit(detail, fleet, h) for h in computers})
            for n, (hosts, software, futs) in list(pending_details.items()):
                if all(f.done() for f in futs.values()):
                    del pending_details[n]
                    details, policy_info = {}, {}
                    for hid, f in futs.items():
                        d = f.result()
                        policy_info.update(d.pop("_policy_info"))
                        details[hid] = d
                    page = {"page": n, "last": len(hosts) < per_page, "hosts": hosts, "software": software,
                            "details": details, "policies": policy_info}
                    done[n] = page
                    if checkpoint:
                        write_json(os.path.join(checkpoint, "pages", f"{n:06d}.json"), page)
        if reports is None:
            if deadline is not None:
                concurrent.futures.wait(report_futures.values(), timeout=max(0, deadline - time.monotonic()))
            else:
                concurrent.futures.wait(report_futures.values())
            if all(f.done() for f in report_futures.values()):
                reports = finish_reports(report_futures, report_info)
                if report_file:
                    write_json(report_file, reports)
    finally:
        page_pool.shutdown(wait=True)
        detail_pool.shutdown(wait=True)
        report_pool.shutdown(wait=False, cancel_futures=True)

    if last is None or any(n not in done for n in range(last + 1)) or reports is None:
        return None
    hosts, software, details, policies = [], {}, {}, {}
    for n in range(last + 1):
        hosts += done[n]["hosts"]
        software.update(done[n]["software"])
        details.update(done[n]["details"])
        policies.update(done[n]["policies"])
    return {
        "schema": 1,
        "generated_at": started.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "fleet_url": fleet.url,
        "fleets": [{"id": f["id"], "name": f["name"]} for f in fleets],
        "hosts": hosts,
        "software_columns": SOFTWARE_COLUMNS,
        "software": software,
        "host_details": details,
        "policies": policies,
        **reports,
    }


def from_env():
    url, token = os.environ.get("FLEET_URL"), os.environ.get("FLEET_API_TOKEN")
    if not url or not token:
        die("FLEET_URL and FLEET_API_TOKEN must be set")
    return Fleet(url, token)


def summary(snap):
    n_sw = sum(len(v) for v in snap["software"].values())
    return (f"hosts={len(snap['hosts'])} software_rows={n_sw} device_health={len(snap['device_health'])} "
            f"os_health={len(snap['os_health'])} ai_tools={len(snap['ai_tools']['hosts'])} "
            f"dex_reports={sum(1 for v in snap['report_info'].values() if v['found'])}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--per-page", type=int, default=20)
    args = ap.parse_args()
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    import build
    snap = collect(from_env(), keep=build.keep_software_row, per_page=args.per_page)
    write_json(args.out, snap)
    print(f"{summary(snap)} bytes={os.path.getsize(args.out)}")


if __name__ == "__main__":
    main()
