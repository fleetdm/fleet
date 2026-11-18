#!/usr/bin/env python3
"""Compute macOS time-to-patch from Fleet's "Get macOS patch history" report.

For every Mac and every Apple security release that applied to it, measures the
days from Apple's release date (SOFA feed) to the first install of a version
that includes the fix. Splits results into before and after the Mac enrolled
in Fleet.

Guide: https://fleetdm.com/guides/measure-macos-time-to-patch

Live:    FLEET_URL=https://fleet.example.com FLEET_API_TOKEN=... \
           ./macos_time_to_patch.py --report-id 123
Offline: ./macos_time_to_patch.py --report-file report.json --hosts-file hosts.json

Standard library only (Python 3.9+).
"""

import argparse
import csv
import json
import math
import os
import statistics
import sys
import urllib.error
import urllib.request
from collections import defaultdict
from datetime import datetime, timezone

SOFA_URL = "https://sofafeed.macadmins.io/v1/macos_data_feed.json"
DAY = 86400


def vtuple(v):
    """'26.5.1' -> (26, 5, 1). Non-numeric parts (betas, RSR suffixes) -> None."""
    try:
        return tuple(int(p) for p in v.strip().split("."))
    except (ValueError, AttributeError):
        return None


def parse_ts(s):
    if not s:
        return None
    return datetime.fromisoformat(s.replace("Z", "+00:00")).timestamp()


# ---------------------------------------------------------------- data loading

def http_json(url, token=None):
    req = urllib.request.Request(url, headers={"User-Agent": "fleet-time-to-patch"})
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return json.load(resp)
    except urllib.error.HTTPError as e:
        hint = " Check FLEET_API_TOKEN." if e.code in (401, 403) else ""
        sys.exit(f"error: {url.split('?')[0]} returned HTTP {e.code}.{hint}")


def load_json_file(path):
    with open(path) as f:
        return json.load(f)


def fetch_hosts(base, token):
    hosts, page = [], 0
    while True:
        data = http_json(f"{base}/api/v1/fleet/hosts?per_page=500&page={page}", token)
        batch = data.get("hosts", [])
        if not batch:
            return hosts
        hosts.extend(batch)
        page += 1


def load_sofa(feed):
    """Return {version_tuple: release} for every macOS security release."""
    releases = {}
    for os_ in feed["OSVersions"]:
        for r in os_["SecurityReleases"]:
            vt = vtuple(r["ProductVersion"])
            if not vt:
                continue
            releases[vt] = {
                "version": r["ProductVersion"],
                "vt": vt,
                "major": vt[0],
                "date": parse_ts(r["ReleaseDate"]),
                "cves": r.get("UniqueCVEsCount") or 0,
                "exploited": len(r.get("ActivelyExploitedCVEs") or []),
            }
    return releases


# ---------------------------------------------------------------- core logic

def parse_installs(raw, releases):
    """Parse 'ver@ts,ver@ts' into (all_installs, released_installs).

    InstallHistory doesn't record build numbers, so a beta of 27.0 looks like
    27.0. Anything installed before Apple publicly released that version, or a
    version Apple never shipped, is pre-release: it still tells us which major
    the host moved to, but it doesn't count as installing the patch.
    """
    every, released = [], []
    for item in (raw or "").split(","):
        ver, _, ts = item.rpartition("@")
        vt, ts = vtuple(ver), int(ts) if ts.isdigit() else None
        if not vt or ts is None:
            continue
        every.append((ts, vt))
        if vt in releases and ts >= releases[vt]["date"]:
            released.append((ts, vt))
    return sorted(every), sorted(released)


def fixes(vt, rel, releases):
    """Does running version `vt` include the fix shipped in release `rel`?

    Same major: any version >= rel. Newer major: only if that version came out
    on or after rel (Apple ships N and N-1 fixes the same day, so 27.0 covers
    26.7 but not 26.7.1).
    """
    if vt[0] == rel["major"]:
        return vt >= rel["vt"]
    if vt[0] > rel["major"] and vt in releases:
        return releases[vt]["date"] >= rel["date"]
    return False


def major_at(t, installs, current):
    """Best guess at which macOS major a host ran at time t."""
    before = [vt for ts, vt in installs if ts < t]
    if before:
        return before[-1][0]
    after = [vt for ts, vt in installs if ts >= t]
    if after:
        return after[0][0]  # factory OS isn't logged; assume first update stayed in-major
    return current[0] if current else None


def evaluate_host(row, releases, window, now, cutover):
    current = vtuple(row.get("current_version") or "")
    start = int(row["history_start"]) if str(row.get("history_start") or "").isdigit() else None
    every, installs = parse_installs(row.get("os_installs"), releases)

    for rel in window:
        if major_at(rel["date"], every, current) != rel["major"]:
            continue  # release was for a different macOS major than the host ran

        rec = {"release": rel["version"], "release_date": rel["date"],
               "cves": rel["cves"], "exploited": rel["exploited"],
               "period": "pre-Fleet" if cutover and rel["date"] < cutover else "Fleet",
               "status": None, "patched_at": None, "days": None}

        if start is None or start > rel["date"]:
            rec["status"] = "no_history"
            yield rec
            continue

        hit = next((ts for ts, vt in installs if ts >= rel["date"] and fixes(vt, rel, releases)), None)
        if hit:
            rec.update(status="patched", patched_at=hit, days=(hit - rel["date"]) / DAY)
        elif current and fixes(current, rel, releases):
            rec["status"] = "patched_untimed"  # e.g. RC build that became final
        else:
            rec.update(status="exposed", days=(now - rel["date"]) / DAY)
        yield rec


# ---------------------------------------------------------------- reporting

def pct(values, p):
    if not values:
        return None
    s = sorted(values)  # nearest-rank
    return s[max(0, math.ceil(p / 100 * len(s)) - 1)]


def summarize(recs, sla):
    patched = [r["days"] for r in recs if r["status"] == "patched"]
    exposed = [r["days"] for r in recs if r["status"] == "exposed"]
    late_open = [d for d in exposed if d > sla]
    on_time = [d for d in patched if d <= sla]
    judged = len(patched) + len(late_open)
    return {
        "measured": len(patched),
        "median_days": round(statistics.median(patched), 1) if patched else None,
        "p90_days": round(pct(patched, 90), 1) if patched else None,
        "mean_days": round(statistics.mean(patched), 1) if patched else None,
        "within_sla_pct": round(100 * len(on_time) / judged, 1) if judged else None,
        "still_exposed": len(exposed),
        "still_exposed_past_sla": len(late_open),
        "oldest_exposure_days": round(max(exposed), 1) if exposed else None,
        "patched_untimed": sum(r["status"] == "patched_untimed" for r in recs),
        "no_history": sum(r["status"] == "no_history" for r in recs),
    }


def fmt(v):
    return "–" if v is None else str(v)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--report-id", type=int, action="append",
                    help="Fleet report ID; repeat for the same report in several fleets")
    ap.add_argument("--report-file", help="Saved JSON from GET /api/v1/fleet/reports/:id/report")
    ap.add_argument("--hosts-file", help="Saved JSON from GET /api/v1/fleet/hosts")
    ap.add_argument("--sofa-file", help="Saved SOFA macOS feed (default: download)")
    ap.add_argument("--days", type=int, default=365, help="Look-back window for releases (default 365)")
    ap.add_argument("--sla-days", type=int, default=14, help="Patch SLA in days (default 14)")
    ap.add_argument("--cutover-date", help="YYYY-MM-DD Fleet go-live for every host "
                    "(default: each host's last_enrolled_at)")
    ap.add_argument("--fleet", help="Only include hosts in the fleet with this name")
    ap.add_argument("--include-zero-cve", action="store_true",
                    help="Also count releases with no published CVEs")
    ap.add_argument("--out-dir", default=".", help="Where to write CSV/JSON output")
    args = ap.parse_args()

    base, token = os.environ.get("FLEET_URL", "").rstrip("/"), os.environ.get("FLEET_API_TOKEN")
    if args.report_file:
        reports = [load_json_file(args.report_file)]
    elif args.report_id and base and token:
        reports = [http_json(f"{base}/api/v1/fleet/reports/{rid}/report", token)
                   for rid in args.report_id]
    else:
        ap.error("pass --report-file, or --report-id with FLEET_URL and FLEET_API_TOKEN set")
    for rep in reports:
        if rep.get("report_clipped"):
            print(f"WARNING: Fleet clipped report {rep.get('report_id') or rep.get('query_id')}; "
                  "some hosts are missing.", file=sys.stderr)
    report = {"results": [row for rep in reports for row in rep.get("results") or []]}

    if args.hosts_file:
        hosts = load_json_file(args.hosts_file)
        hosts = hosts.get("hosts", hosts) if isinstance(hosts, dict) else hosts
    elif base and token:
        hosts = fetch_hosts(base, token)
    else:
        hosts = []
    hosts = {h["id"]: h for h in hosts}

    releases = load_sofa(load_json_file(args.sofa_file) if args.sofa_file else http_json(SOFA_URL))
    now = datetime.now(timezone.utc).timestamp()
    since = now - args.days * DAY
    window = sorted((r for r in releases.values()
                     if since <= r["date"] <= now and (args.include_zero_cve or r["cves"] > 0)),
                    key=lambda r: r["date"])
    global_cutover = parse_ts(args.cutover_date + "T00:00:00Z") if args.cutover_date else None

    recs = []
    for row in report.get("results", []):
        hid = row["host_id"]
        host = hosts.get(hid, {})
        fleet = host.get("fleet_name") or host.get("team_name")
        if args.fleet and fleet != args.fleet:
            continue
        cutover = global_cutover or parse_ts(host.get("last_enrolled_at"))
        for rec in evaluate_host(row.get("columns", {}), releases, window, now, cutover):
            rec.update(host_id=hid, host_name=host.get("display_name") or row.get("host_name"),
                       fleet=fleet)
            recs.append(rec)

    os.makedirs(args.out_dir, exist_ok=True)
    iso = lambda t: datetime.fromtimestamp(t, timezone.utc).strftime("%Y-%m-%d %H:%M") if t else ""
    with open(os.path.join(args.out_dir, "time_to_patch_by_host.csv"), "w", newline="") as f:
        cols = ["host_id", "host_name", "fleet", "release", "release_date", "cves", "exploited",
                "period", "status", "patched_at", "days"]
        w = csv.DictWriter(f, fieldnames=cols)
        w.writeheader()
        for r in recs:
            w.writerow({**{c: r[c] for c in cols}, "release_date": iso(r["release_date"])[:10],
                        "patched_at": iso(r["patched_at"]),
                        "days": "" if r["days"] is None else round(r["days"], 1)})

    by_period, by_month = defaultdict(list), defaultdict(list)
    for r in recs:
        by_period[r["period"]].append(r)
        by_month[(iso(r["release_date"])[:7], r["period"])].append(r)
    summary = {
        "generated": iso(now), "sla_days": args.sla_days,
        "hosts": len({r["host_id"] for r in recs}),
        "releases": [r["version"] for r in window],
        "overall": summarize(recs, args.sla_days),
        "by_period": {p: summarize(v, args.sla_days) for p, v in sorted(by_period.items())},
        "by_month": {f"{m} {p}": summarize(v, args.sla_days) for (m, p), v in sorted(by_month.items())},
    }
    with open(os.path.join(args.out_dir, "time_to_patch_summary.json"), "w") as f:
        json.dump(summary, f, indent=2)

    print(f"macOS time-to-patch: {summary['hosts']} hosts, {len(window)} security releases, "
          f"SLA {args.sla_days} days\n")
    print("| Scope | Measured | Median | p90 | Within SLA | Still exposed (past SLA) | No history |")
    print("|---|---|---|---|---|---|---|")
    rows = [("All", summary["overall"])] + list(summary["by_period"].items())
    for name, s in rows:
        sla_pct = f"{s['within_sla_pct']}%" if s["within_sla_pct"] is not None else "–"
        print(f"| {name} | {s['measured']} | {fmt(s['median_days'])} | {fmt(s['p90_days'])} | {sla_pct} "
              f"| {s['still_exposed']} ({s['still_exposed_past_sla']}) | {s['no_history']} |")
    print(f"\nWrote {args.out_dir}/time_to_patch_by_host.csv and time_to_patch_summary.json")


if __name__ == "__main__":
    main()
