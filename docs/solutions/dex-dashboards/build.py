#!/usr/bin/env python3
"""Build a DEX dashboard page from a collect.py snapshot.

  build.py --page lifecycle|software|patch --snapshot snap.json \
           --templates DIR --history in.json --out-html index.html --out-history out.json

The history file carries state between daily runs (strain days, version
first-seen dates). Pass a missing or empty path on the first run.
"""
import argparse
import datetime as dt
import json
import os
import re
import statistics
import sys

STALE_DAYS = 30
WARRANTY_VITAL = os.environ.get("DEX_WARRANTY_VITAL", "AppleCare warranty expiration")
PATCH_SLA_DAYS = 14
IDLE_DAYS = 90
MIN_INSTALLS = 3

# Apple silicon release years by generation and tier. Intel Macs are
# treated as older than any Apple silicon.
CURRENT_APPLE_GEN = 5
CHIP_YEARS = {
    1: {"base": 2020, "pro": 2021, "max": 2021, "ultra": 2022},
    2: {"base": 2022, "pro": 2023, "max": 2023, "ultra": 2023},
    3: {"base": 2023, "pro": 2023, "max": 2023, "ultra": 2025},
    4: {"base": 2024, "pro": 2024, "max": 2024},
    5: {"base": 2025},
}

SUPPORT_RE = re.compile(
    r"(uninstall|helper|updater|update assistant|installer|diagnostic|crash ?report|"
    r"pluginservice|launchagent|login item|\bagent\b|\bdaemon\b|\bservice\b)", re.I)
# Agents IT deploys to every Mac. They are opened by the system, not by people,
# so they are never seats to reclaim.
MANAGED_AGENTS = {
    "fleet desktop", "nudge", "santa", "okta verify", "falcon", "sentinelone",
    "jamf connect", "self service", "munki", "managed software center",
}
VENDOR_NAMES = {
    "microsoft": "Microsoft", "adobe": "Adobe", "google": "Google", "jetbrains": "JetBrains",
    "docker": "Docker", "tinyspeck": "Slack", "zoom": "Zoom", "agilebits": "1Password",
    "1password": "1Password", "okta": "Okta", "github": "GitHub", "mozilla": "Mozilla",
    "anthropic": "Anthropic", "openai": "OpenAI", "figma": "Figma", "parallels": "Parallels",
    "vmware": "VMware", "grammarly": "Grammarly", "logi": "Logitech", "logitech": "Logitech",
    "sublimetext": "Sublime HQ", "notion": "Notion", "linear": "Linear", "loom": "Loom",
    "postmanlabs": "Postman", "electron": "Electron", "utmapp": "UTM", "googlecode": "iTerm2",
    "brave": "Brave", "operasoftware": "Opera", "vivaldi": "Vivaldi", "spotify": "Spotify",
    "dropbox": "Dropbox", "getdropbox": "Dropbox", "box": "Box", "cisco": "Cisco",
    "webex": "Webex", "amazon": "Amazon", "amazonaws": "AWS", "hnc": "Discord",
    "anysphere": "Cursor", "warp": "Warp", "raycast": "Raycast", "obsproject": "OBS",
    "videolan": "VideoLAN", "wireshark": "Wireshark", "iterm2": "iTerm2", "macpaw": "MacPaw",
    "digidna": "DigiDNA", "kandji": "Kandji", "jamf": "Jamf", "fleetdm": "Fleet",
    "blackmagic-design": "Blackmagic Design", "winehq": "WineHQ", "macadmins": "Mac Admins",
    "northpolesec": "North Pole Security", "simplemdm": "SimpleMDM",
}
WRAPPER_VENDORS = {"todesktop", "electron", "github.electron", "nativefier"}


# ---------------------------------------------------------------- helpers

def parse_ts(s):
    if not s:
        return None
    try:
        return dt.datetime.strptime(s[:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dt.timezone.utc)
    except ValueError:
        return None


def days_between(a, b):
    return (b - a).total_seconds() / 86400


def host_name(h):
    return h.get("display_name") or h.get("computer_name") or h.get("hostname") or f"Host {h['id']}"


def vkey(v):
    """Natural sort key for version strings like '7.2.2 (88465)' or 'Build 4215'."""
    parts = re.findall(r"\d+|[A-Za-z]+", v or "")
    return [(0, int(p), "") if p.isdigit() else (1, 0, p.lower()) for p in parts]


def vcmp(a, b):
    ka, kb = vkey(a), vkey(b)
    return (ka > kb) - (ka < kb)


def pct(n, d):
    return round(100 * n / d) if d else None


def quantile(xs, q):
    xs = sorted(xs)
    if not xs:
        return None
    i = (len(xs) - 1) * q
    lo, hi = int(i), min(int(i) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (i - lo)


def parse_chip(brand):
    b = (brand or "").strip()
    m = re.search(r"\bApple\s+M(\d+)(?:\s+(Pro|Max|Ultra))?\b", b, re.I)
    if m:
        gen, tier = int(m.group(1)), (m.group(2) or "base").lower()
        years = CHIP_YEARS.get(gen, {})
        return {
            "vendor": "apple", "gen": gen, "label": f"M{gen}" + ("" if tier == "base" else " " + tier.title()),
            "family": f"M{gen}", "year": years.get(tier) or years.get("base"),
            "gens_behind": max(0, CURRENT_APPLE_GEN - gen),
        }
    if re.search(r"\bintel\b", b, re.I):
        return {"vendor": "intel", "gen": None, "label": "Intel", "family": "Intel", "year": 2020,
                "gens_behind": CURRENT_APPLE_GEN + 1}
    return None


def host_map(hosts, ids):
    return {hid: {"n": host_name(hosts[hid]), "f": hosts[hid].get("team_name") or "Unassigned"} for hid in ids}


def load_json(path, default):
    if not path or not os.path.exists(path) or os.path.getsize(path) == 0:
        return default
    with open(path) as f:
        return json.load(f)


def trim_daily(daily, keep_days, today):
    cutoff = (today - dt.timedelta(days=keep_days)).isoformat()
    return {k: v for k, v in daily.items() if k >= cutoff}


# ---------------------------------------------------------------- lifecycle

def strain_today(swap, uptime_days):
    """swap_pressure tiers count swap-ins since boot, so weigh them by uptime."""
    if uptime_days is None:
        return False, False
    pressured = (swap == "severe" and uptime_days < 7) or (swap == "elevated" and uptime_days < 2)
    severe = swap == "severe" and uptime_days < 3
    return pressured, severe


def lifecycle_verdict(m):
    old = m["gens_behind"] is not None and m["gens_behind"] >= 3
    if m["battery_service"]:
        return "refresh" if old else "battery"
    rep, prs = m["days_reporting"], m["days_pressured"]
    sustained = rep >= 5 and prs >= max(3, rep * 0.5)
    if sustained:
        return "refresh" if old else "investigate"
    if sustained or m["pressured_now"] or m["battery_score"] == "degraded":
        return "watch"
    return "defer" if old else "healthy"


def build_lifecycle(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    hist.setdefault("schema", 1)
    hist.setdefault("since", today.isoformat())
    strain = hist.setdefault("strain", {})
    summary_hist = hist.setdefault("summary", {})
    details = snap.get("host_details", {})
    dh_rows = snap.get("device_health", {})

    macs = []
    for h in snap["hosts"]:
        if h["platform"] != "darwin":
            continue
        hid = str(h["id"])
        dh = dh_rows.get(hid) or {}
        det = details.get(hid) or {}
        chip = parse_chip(h.get("cpu_brand") or dh.get("cpu_brand"))
        mkt = det.get("hardware_marketing_name") or ""
        ym = re.search(r"\b(20\d\d)\b", mkt)
        year = int(ym.group(1)) if ym else (chip["year"] if chip else None)
        seen = parse_ts(h.get("seen_time"))
        seen_days = days_between(seen, now) if seen else None
        uptime_days = (h.get("uptime") or 0) / 1e9 / 86400 if h.get("uptime") else None
        fetched = parse_ts(dh.get("last_fetched"))
        dh_fresh = bool(fetched and days_between(fetched, now) <= 1.5)
        pressured, severe = strain_today(dh.get("swap_pressure"), uptime_days) if dh_fresh else (False, False)

        days = strain.setdefault(hid, {})
        if dh_fresh:
            days[today.isoformat()] = [int(pressured), int(severe)]
        cutoff = (today - dt.timedelta(days=30)).isoformat()
        recent = {d: v for d, v in days.items() if d > cutoff}
        strain[hid] = {d: v for d, v in days.items() if d > (today - dt.timedelta(days=45)).isoformat()}

        batteries = det.get("batteries") or []
        fleet_batt = batteries[0].get("health") if batteries else None
        score = dh.get("battery_health_score") or None
        battery_service = score == "replace" or (fleet_batt or "").lower().startswith("service")
        mem = h.get("memory") or 0
        ram = round(mem / 2**30) if mem else None
        warranty = (det.get("vitals") or {}).get(WARRANTY_VITAL) or None
        m = {
            "id": h["id"], "name": host_name(h), "fleet": h.get("team_name") or "Unassigned",
            "model": mkt or h.get("hardware_model") or "", "model_id": h.get("hardware_model") or "",
            "serial": h.get("hardware_serial") or "",
            "chip": chip["label"] if chip else (h.get("cpu_brand") or "Unknown"),
            "chip_family": chip["family"] if chip else "Unknown",
            "gens_behind": chip["gens_behind"] if chip else None,
            "year": year, "age": (today.year - year) if year else None,
            "ram": ram, "os": h.get("os_version") or "",
            "battery_pct": float(dh["battery_health_pct"]) if dh.get("battery_health_pct") else None,
            "battery_cycles": int(dh["battery_cycles"]) if (dh.get("battery_cycles") or "").isdigit() else None,
            "battery_score": score, "battery_fleet": fleet_batt, "battery_service": battery_service,
            "swap": dh.get("swap_pressure") if dh_fresh else None,
            "uptime_days": round(uptime_days, 1) if uptime_days is not None else None,
            "pressured_now": pressured,
            "days_reporting": len(recent),
            "days_pressured": sum(v[0] for v in recent.values()),
            "days_severe": sum(v[1] for v in recent.values()),
            "warranty": warranty,
            "warranty_days": (dt.date.fromisoformat(warranty) - today).days if warranty and re.match(r"\d{4}-\d\d-\d\d$", warranty) else None,
            "disk_free_pct": h.get("percent_disk_space_available"),
            "seen_days": round(seen_days, 1) if seen_days is not None else None,
            "has_health": dh_fresh,
        }
        stale = seen_days is None or seen_days > STALE_DAYS
        m["verdict"] = "stale" if stale else lifecycle_verdict(m)
        macs.append(m)

    # Hosts that are gone from Fleet stop accruing history.
    live = {str(m["id"]) for m in macs}
    for hid in list(strain):
        if hid not in live:
            del strain[hid]

    active = [m for m in macs if m["verdict"] != "stale"]
    counts = {k: sum(1 for m in macs if m["verdict"] == k)
              for k in ["refresh", "battery", "investigate", "watch", "defer", "healthy", "stale"]}
    flagged = [m for m in active if m["verdict"] not in ("healthy",)]
    signals = [
        {"label": "Chip 3+ generations old", "count": sum(1 for m in active if (m["gens_behind"] or 0) >= 3)},
        {"label": "Sustained memory strain", "count": sum(1 for m in active if m["days_reporting"] >= 5 and m["days_pressured"] >= max(3, m["days_reporting"] * .5))},
        {"label": "16 GB of RAM or less", "count": sum(1 for m in active if m["ram"] and m["ram"] <= 16)},
        {"label": "Battery degraded or needs service", "count": sum(1 for m in active if m["battery_service"] or m["battery_score"] == "degraded")},
        {"label": "Warranty ends within 90 days", "count": sum(1 for m in active if m["warranty_days"] is not None and 0 <= m["warranty_days"] <= 90)},
    ]
    ages = [m["age"] for m in active if m["age"] is not None]
    summary_hist[today.isoformat()] = {k: counts[k] for k in counts}
    hist["summary"] = trim_daily(summary_hist, 400, today)
    days_collected = len({d for v in strain.values() for d in v})
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "history_since": hist["since"], "history_days": days_collected,
        "macs": macs, "current_gen": CURRENT_APPLE_GEN, "stale_days": STALE_DAYS, "warranty_vital": WARRANTY_VITAL,
        "trend": [{"date": d, **v} for d, v in sorted(hist["summary"].items())],
    }


# ---------------------------------------------------------------- software

def vendor_of(bid, name):
    parts = (bid or "").lower().split(".")
    if len(parts) < 2:
        return name
    tld = parts[0] in ("com", "org", "net", "io", "us", "de", "co", "app", "dev", "ch", "uk", "fr", "jp", "se", "nl", "at", "be", "ru")
    org = parts[1] if tld else parts[0]
    if org in WRAPPER_VENDORS or ".".join(parts[1:3]) in WRAPPER_VENDORS:
        return name
    # com.github.<org>.<app> names the GitHub org that publishes the app.
    if org == "github" and tld and len(parts) >= 4:
        org = parts[2]
    if org in VENDOR_NAMES:
        return VENDOR_NAMES[org]
    words = re.split(r"[-_]", org)
    return " ".join(w[:1].upper() + w[1:] for w in words if w)


def tracked_mac_apps(rows):
    """Top-level third-party apps in /Applications, one entry per bundle."""
    out = {}
    for name, ver, source, bid, last, path in rows:
        if source != "apps" or not path.startswith("/Applications/") or path.count(".app") != 1:
            continue
        if bid.startswith("com.apple."):
            continue
        key = bid or name
        prev = out.get(key)
        if prev is None or vcmp(ver, prev[1]) > 0:
            out[key] = (name, ver, bid, last, path)
    return out


def app_kind(name, bid):
    b = (bid or "").lower()
    # Drive for desktop and Chrome install launchers that just open a web page.
    if ".shortcuts." in b or ".chrome.app." in b:
        return "support"
    if name.lower() in MANAGED_AGENTS:
        return "agent"
    if SUPPORT_RE.search(name):
        return "support"
    return "app"


def build_software(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    hist.setdefault("schema", 1)
    hist.setdefault("since", today.isoformat())
    summary_hist = hist.setdefault("summary", {})
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    active = []
    for hid, h in hosts.items():
        seen = parse_ts(h.get("seen_time"))
        if h["platform"] == "darwin" and seen and days_between(seen, now) <= STALE_DAYS:
            active.append(hid)

    apps = {}
    for hid in active:
        for key, (name, ver, bid, last, path) in tracked_mac_apps(snap["software"].get(hid, [])).items():
            a = apps.setdefault(key, {"key": key, "names": {}, "bid": bid, "hosts": []})
            a["names"][name] = a["names"].get(name, 0) + 1
            opened = parse_ts(last) if last else None
            ds = round(days_between(opened, now)) if opened else None
            a["hosts"].append([int(hid), ver, ds])

    rows = []
    for a in apps.values():
        name = max(a["names"], key=a["names"].get)
        hs = a["hosts"]
        idle = [x for x in hs if x[2] is None or x[2] >= IDLE_DAYS]
        never = [x for x in hs if x[2] is None]
        opened_days = [x[2] for x in hs if x[2] is not None]
        rows.append({
            "key": a["key"], "name": name, "bid": a["bid"], "vendor": vendor_of(a["bid"], name),
            "kind": app_kind(name, a["bid"]), "installs": len(hs), "idle": len(idle), "never": len(never),
            "pct_idle": pct(len(idle), len(hs)),
            "median_days": statistics.median(opened_days) if opened_days else None,
            "hosts": sorted(hs, key=lambda x: (x[2] is not None, -(x[2] or 0))),
        })
    seats = [r for r in rows if r["kind"] == "app"]
    total = sum(r["installs"] for r in seats)
    idle = sum(r["idle"] for r in seats)
    vendors = {}
    for r in seats:
        v = vendors.setdefault(r["vendor"], {"vendor": r["vendor"], "apps": 0, "seats": 0, "idle": 0})
        v["apps"] += 1
        v["seats"] += r["installs"]
        v["idle"] += r["idle"]
    for v in vendors.values():
        v["pct_idle"] = pct(v["idle"], v["seats"])
    summary_hist[today.isoformat()] = {"seats": total, "idle": idle}
    hist["summary"] = trim_daily(summary_hist, 400, today)
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "hosts": host_map(hosts, active),
        "apps": [{"key": r["key"], "name": r["name"], "vendor": r["vendor"], "kind": r["kind"], "h": r["hosts"]}
                 for r in sorted(rows, key=lambda r: -r["idle"])],
        "idle_days": IDLE_DAYS, "min_installs": MIN_INSTALLS, "stale_days": STALE_DAYS,
        "trend": [{"date": d, **v} for d, v in sorted(hist["summary"].items())],
    }


# ---------------------------------------------------------------- patch velocity

def patch_titles(snap, hid, h):
    """Version per tracked title for one host: macOS apps plus the OS."""
    out = {}
    if h["platform"] == "darwin":
        for key, (name, ver, bid, last, path) in tracked_mac_apps(snap["software"].get(hid, [])).items():
            if app_kind(name, bid) == "support":
                continue
            out["app:" + key] = (name, ver)
        osv = (h.get("os_version") or "").replace("macOS ", "").strip()
        if osv:
            out["os:macOS"] = ("macOS", osv)
    return out


def newest_release(vers, betas):
    """Highest version at least two hosts run, so a lone test build doesn't set the bar."""
    release = [v for v in vers if v not in betas] or list(vers)
    counts = {}
    for v in release:
        counts[v] = counts.get(v, 0) + 1
    shared = [v for v, c in counts.items() if c >= 2]
    return max(shared or release, key=vkey)


def build_patch(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    D = today.isoformat()
    first_run = not hist.get("hosts")
    hist.setdefault("schema", 1)
    hist.setdefault("since", D)
    versions = hist.setdefault("versions", {})
    names = hist.setdefault("names", {})
    state = hist.setdefault("hosts", {})
    events = hist.setdefault("events", [])
    summary_hist = hist.setdefault("summary", {})
    hosts = {str(h["id"]): h for h in snap["hosts"]}

    betas = {(r.get("os_version") or "") for r in snap.get("os_health", {}).values()
             if re.search(r"[a-z]$", r.get("os_build") or "")}

    current = {}
    for hid, h in hosts.items():
        titles = patch_titles(snap, hid, h)
        current[hid] = titles
        prev_host = state.get(hid)
        new_host = {}
        for key, (name, ver) in titles.items():
            names[key] = name
            vs = versions.setdefault(key, {})
            if ver not in vs:
                vs[ver] = None if first_run else D
            prev = (prev_host or {}).get(key)
            if prev is None:
                new_host[key] = [ver, D]
            elif prev[0] != ver:
                if vcmp(ver, prev[0]) > 0 and ver not in betas:
                    events.append([key, prev[0], ver, int(hid), vs.get(ver), D])
                new_host[key] = [ver, D]
            else:
                new_host[key] = prev
        state[hid] = new_host
    for hid in list(state):
        if hid not in hosts:
            del state[hid]
    cutoff180 = (today - dt.timedelta(days=180)).isoformat()
    hist["events"] = events = [e for e in events if e[5] >= cutoff180]

    # Per-title installs on hosts that still check in. The page aggregates these
    # (per fleet) against the fleet-wide newest release.
    active = [hid for hid, h in hosts.items()
              if parse_ts(h.get("seen_time")) and days_between(parse_ts(h["seen_time"]), now) <= STALE_DAYS]
    by_title = {}
    for hid in active:
        for key, (name, ver) in current[hid].items():
            by_title.setdefault(key, []).append([int(hid), ver])
    recent = (today - dt.timedelta(days=45)).isoformat()
    titles, index = [], {}
    for key, hv in sorted(by_title.items()):
        if len(hv) < 2:
            continue
        newest = newest_release([v for _, v in hv], betas)
        index[key] = len(titles)
        titles.append({
            "key": key, "name": names.get(key, key), "type": "os" if key.startswith("os:") else "app",
            "newest": newest, "newest_seen": versions.get(key, {}).get(newest),
            "i": hv,
            "fresh": sorted([[v, seen] for v, seen in versions.get(key, {}).items()
                             if seen and seen >= recent and v not in betas], key=lambda x: x[1]),
        })

    # Measured latency: only updates to versions first seen after the baseline.
    cutoff90 = (today - dt.timedelta(days=90)).isoformat()
    measured = []
    for key, frm, to, hid, first_seen, applied in events:
        if applied < cutoff90 or not first_seen or key not in index:
            continue
        lag = (dt.date.fromisoformat(applied) - dt.date.fromisoformat(first_seen)).days
        measured.append([index[key], hid, frm, to, first_seen, applied, lag])

    installs = on_newest = 0
    for t in titles:
        if t["type"] == "app" and len(t["i"]) >= MIN_INSTALLS:
            installs += len(t["i"])
            on_newest += sum(1 for _, v in t["i"] if vcmp(v, t["newest"]) >= 0)
    summary_hist[D] = {"pct_current": pct(on_newest, installs), "events": len(events)}
    hist["summary"] = trim_daily(summary_hist, 400, today)
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "history_since": hist["since"], "sla_days": PATCH_SLA_DAYS, "stale_days": STALE_DAYS,
        "min_installs": MIN_INSTALLS, "hosts": host_map(hosts, active),
        "titles": titles, "events": measured, "betas": sorted(betas),
        "trend": [{"date": d, **v} for d, v in sorted(hist["summary"].items())],
    }


# ---------------------------------------------------------------- render

BUILDERS = {"lifecycle": build_lifecycle, "software": build_software, "patch": build_patch}


def render(templates, page, data):
    def read(name):
        with open(os.path.join(templates, name)) as f:
            return f.read()
    page_html = read(f"{page}.html")
    blob = json.dumps(data, separators=(",", ":")).replace("</", "<\\/")
    return (page_html
            .replace("/*__BASE_CSS__*/", read("base.css"))
            .replace("/*__BASE_JS__*/", read("base.js"))
            .replace("/*__DATA__*/null", blob))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--page", required=True, choices=sorted(BUILDERS))
    ap.add_argument("--snapshot", required=True)
    ap.add_argument("--templates", required=True)
    ap.add_argument("--history")
    ap.add_argument("--out-html", required=True)
    ap.add_argument("--out-history", required=True)
    args = ap.parse_args()
    snap = load_json(args.snapshot, None)
    if not snap or snap.get("schema") != 1:
        sys.exit("FATAL: snapshot missing or wrong schema")
    hist = load_json(args.history, {})
    data = BUILDERS[args.page](snap, hist)
    with open(args.out_html, "w") as f:
        f.write(render(args.templates, args.page, data))
    with open(args.out_history, "w") as f:
        json.dump(hist, f, separators=(",", ":"))
    print(f"{args.page}: html={os.path.getsize(args.out_html)}B history={os.path.getsize(args.out_history)}B")


if __name__ == "__main__":
    main()
