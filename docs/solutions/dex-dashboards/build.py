#!/usr/bin/env python3
"""Build a DEX dashboard page from a collect.py snapshot.

  build.py --page lifecycle|software|patch --snapshot snap.json \
           --templates DIR --history in.json --out-html index.html --out-history out.json

The history file carries state between daily runs (strain days, version
first-seen dates). Pass a missing or empty path on the first run.
"""
import argparse
import datetime as dt
import html
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
# Intel and AMD CPUs count as old this many years after their release year.
OLD_CPU_YEARS = 4
NOT_COMPUTERS = {"ios", "ipados", "android", "chrome"}

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
# First release year of each Intel Core generation and AMD Ryzen series.
# A computer can't be newer than its CPU, so these are lower bounds on age.
INTEL_CORE_YEARS = {4: 2013, 5: 2015, 6: 2015, 7: 2016, 8: 2017, 9: 2018, 10: 2019,
                    11: 2020, 12: 2021, 13: 2022, 14: 2023}
INTEL_ULTRA_YEARS = {1: 2023, 2: 2024}
RYZEN_YEARS = {1: 2017, 2: 2018, 3: 2019, 4: 2020, 5: 2020, 6: 2022, 7: 2022, 8: 2024, 9: 2024}

VIRTUAL_RE = re.compile(
    r"qemu|vmware|virtualbox|innotek|parallels|\bxen\b|\bkvm\b|bochs|hyper-v|virtual machine|"
    r"virtual cpu|\(virtual\)|virtualmac|droplet|digitalocean|amazon ec2|google compute", re.I)

SUPPORT_RE = re.compile(
    r"(uninstall|helper|updater|update assistant|installer|diagnostic|crash ?report|"
    r"pluginservice|launchagent|login item|\bagent\b|\bdaemon\b|\bservice\b)", re.I)
# Agents IT deploys everywhere. They are opened by the system, not by people,
# so they are never seats to reclaim.
MANAGED_AGENTS = {
    "fleet desktop", "fleet osquery", "fleet-osquery", "orbit", "nudge", "santa", "okta verify",
    "falcon", "crowdstrike windows sensor", "sentinelone", "sentinel agent", "jamf connect",
    "self service", "munki", "managed software center", "company portal",
}
# Windows components, runtimes, and drivers. The Windows counterpart of
# leaving Apple's own apps out on macOS.
WINDOWS_BUILTIN_RE = re.compile(
    r"^(microsoft\.|microsoftwindows\.|windows\.|windows |microsoft edge|microsoft store|"
    r"microsoft onedrive|microsoft update|microsoft visual c\+\+|microsoft \.net|"
    r"microsoft windows desktop runtime|microsoft gameinput|microsoft teams meeting add-in|"
    r"[0-9a-f]{8}-[0-9a-f]{4}-)|"
    r"\.exe$|webview2|redistributable|runtime\b|\bsdk\b|\bdriver|chipset|firmware|realtek|"
    r"^intel\(r\)|^nvidia|^amd |thunderbolt|bluetooth|health tools|pc health check", re.I)
# Microsoft publishes Windows' own apps too, so only its licensed products count.
MICROSOFT_SEATS_RE = re.compile(
    r"^microsoft (365|office|teams|visio|project|visual studio|power bi|sql server management)|"
    r"^(ms)?teams$|visual studio code", re.I)
# Linux packages are mostly the OS itself, so only these desktop apps count
# as seats. Package name -> (display name, vendor).
LINUX_DESKTOP_APPS = {
    "google-chrome-stable": ("Google Chrome", "Google"), "chromium": ("Chromium", "Chromium"),
    "chromium-browser": ("Chromium", "Chromium"), "firefox": ("Mozilla Firefox", "Mozilla"),
    "microsoft-edge-stable": ("Microsoft Edge", "Microsoft"), "brave-browser": ("Brave", "Brave"),
    "code": ("Visual Studio Code", "Microsoft"), "cursor": ("Cursor", "Cursor"),
    "sublime-text": ("Sublime Text", "Sublime HQ"), "slack-desktop": ("Slack", "Slack"),
    "slack": ("Slack", "Slack"), "zoom": ("Zoom", "Zoom"), "teams-for-linux": ("Teams for Linux", "Teams for Linux"),
    "1password": ("1Password", "1Password"), "docker-desktop": ("Docker Desktop", "Docker"),
    "spotify-client": ("Spotify", "Spotify"), "discord": ("Discord", "Discord"),
    "signal-desktop": ("Signal", "Signal"), "postman": ("Postman", "Postman"),
    "dbeaver-ce": ("DBeaver", "DBeaver"), "obsidian": ("Obsidian", "Obsidian"),
    "insomnia": ("Insomnia", "Kong"), "vlc": ("VLC", "VideoLAN"), "gimp": ("GIMP", "GIMP"),
    "inkscape": ("Inkscape", "Inkscape"), "libreoffice-core": ("LibreOffice", "The Document Foundation"),
    "thunderbird": ("Thunderbird", "Mozilla"), "wireshark": ("Wireshark", "Wireshark"),
    "virtualbox": ("VirtualBox", "Oracle"), "remmina": ("Remmina", "Remmina"),
    "keepassxc": ("KeePassXC", "KeePassXC"), "github-desktop": ("GitHub Desktop", "GitHub"),
    "warp-terminal": ("Warp", "Warp"), "claude-desktop": ("Claude", "Anthropic"),
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
    "slacktechnologies": "Slack", "zoomvideocommunications": "Zoom", "zoomcommunications": "Zoom",
    "googleinc": "Google", "mozillafoundation": "Mozilla", "anthropicpbc": "Anthropic",
    "adobesystems": "Adobe", "dockerinc": "Docker", "githubinc": "GitHub", "jetbrainss.r.o": "JetBrains", "jetbrainssro": "JetBrains",
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


def platform_of(h):
    """macos, windows, or linux. None for phones, tablets, and Chromebooks."""
    p = h.get("platform") or ""
    if p == "darwin":
        return "macos"
    if p == "windows":
        return "windows"
    if p in NOT_COMPUTERS or not p:
        return None
    return "linux"


def is_virtual(h):
    text = " ".join(str(h.get(k) or "") for k in ("hardware_vendor", "hardware_model", "cpu_brand"))
    return bool(VIRTUAL_RE.search(text))


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


def software_rows(snap, hid):
    """Software rows as (name, version, source, bundle_id, last_opened, path, vendor)."""
    for r in snap["software"].get(hid, []):
        yield (*r[:6], r[6] if len(r) > 6 else "")


def intel_core_gen(brand):
    if (m := re.search(r"(\d+)th Gen", brand, re.I)):
        return int(m.group(1))
    if not (m := re.search(r"\bi[3579]-(\d{4,5})", brand)):
        return None
    digits = m.group(1)
    # 10th gen and later model numbers start with the generation (i7-1365U, i7-12700).
    if digits[0] == "1" and 10 <= int(digits[:2]) <= 14:
        return int(digits[:2])
    return int(digits[0])


def parse_cpu(brand, this_year):
    """CPU label, family, release year, and whether it's old enough to refresh."""
    b = re.sub(r"\s+", " ", (brand or "").replace("(R)", "").replace("(TM)", "")).strip()
    m = re.search(r"\bApple M(\d+)(?: (Pro|Max|Ultra))?\b", b, re.I)
    if m:
        gen, tier = int(m.group(1)), (m.group(2) or "base").lower()
        years = CHIP_YEARS.get(gen, {})
        behind = max(0, CURRENT_APPLE_GEN - gen)
        return {"label": f"M{gen}" + ("" if tier == "base" else " " + tier.title()), "family": f"Apple M{gen}",
                "year": years.get(tier) or years.get("base"), "gens_behind": behind, "old": behind >= 3}
    year, family, label = None, None, b or "Unknown"
    if re.search(r"\bintel\b", b, re.I):
        if (m := re.search(r"Core Ultra \d+ (\d)\d\d", b, re.I)):
            series = int(m.group(1))
            year, family = INTEL_ULTRA_YEARS.get(series), f"Intel Core Ultra {series}00"
        elif (gen := intel_core_gen(b)):
            year, family = INTEL_CORE_YEARS.get(gen), f"Intel Core {gen}th gen"
        elif (m := re.search(r"\bN(\d)(\d)\d\b", b)):
            year, family = (2025 if m.group(2) == "5" else 2023), "Intel N-series"
        else:
            family = "Intel (other)"
        label = re.sub(r"^\d+th Gen ", "", b)
    elif re.search(r"\bryzen\b", b, re.I):
        if re.search(r"Ryzen AI Max", b, re.I):
            year, family = 2025, "AMD Ryzen AI Max"
        elif (m := re.search(r"Ryzen AI \d+(?: HX)?(?: PRO)? (\d)\d\d", b, re.I)):
            year, family = 2024, f"AMD Ryzen AI {m.group(1)}00"
        elif (m := re.search(r"Ryzen (?:Threadripper )?(?:PRO )?\d+ (?:PRO )?(\d)\d{3}", b, re.I)):
            series = int(m.group(1))
            year, family = RYZEN_YEARS.get(series), f"AMD Ryzen {series}000"
        else:
            family = "AMD Ryzen (other)"
        label = re.sub(r" w/ Radeon.*$| with Radeon.*$| \d+-Core Processor$", "", b, flags=re.I)
    elif re.search(r"snapdragon", b, re.I):
        year, family = 2024, "Snapdragon X"
    if family is None:
        return None
    return {"label": label, "family": family, "year": year, "gens_behind": None,
            "old": bool(year and this_year - year >= OLD_CPU_YEARS)}


def host_map(hosts, ids):
    return {hid: {"n": host_name(hosts[hid]), "f": hosts[hid].get("team_name") or "Unassigned",
                  "p": platform_of(hosts[hid])} for hid in ids}


def is_active(h, now):
    seen = parse_ts(h.get("seen_time"))
    return bool(seen and days_between(seen, now) <= STALE_DAYS)


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
    old = m["old"]
    if m["battery_service"]:
        return "refresh" if old else "battery"
    if m["windows10"]:
        return "refresh" if old else "investigate"
    rep, prs = m["days_reporting"], m["days_pressured"]
    sustained = rep >= 5 and prs >= max(3, rep * 0.5)
    if sustained:
        return "refresh" if old else "investigate"
    if m["pressured_now"] or m["battery_score"] == "degraded":
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

    devices = []
    for h in snap["hosts"]:
        plat = platform_of(h)
        if not plat:
            continue
        hid = str(h["id"])
        dh = (dh_rows.get(hid) or {}) if plat == "macos" else {}
        det = details.get(hid) or {}
        cpu = parse_cpu(h.get("cpu_brand") or dh.get("cpu_brand"), today.year)
        mkt = det.get("hardware_marketing_name") or ""
        ym = re.search(r"\b(20\d\d)\b", mkt)
        year = int(ym.group(1)) if ym else (cpu["year"] if cpu else None)
        seen = parse_ts(h.get("seen_time"))
        seen_days = days_between(seen, now) if seen else None
        uptime_days = (h.get("uptime") or 0) / 1e9 / 86400 if h.get("uptime") else None
        fetched = parse_ts(dh.get("last_fetched"))
        dh_fresh = bool(fetched and days_between(fetched, now) <= 1.5)
        pressured, severe = strain_today(dh.get("swap_pressure"), uptime_days) if dh_fresh else (False, False)

        days = strain.setdefault(hid, {}) if plat == "macos" else {}
        if dh_fresh:
            days[today.isoformat()] = [int(pressured), int(severe)]
        cutoff = (today - dt.timedelta(days=30)).isoformat()
        recent = {d: v for d, v in days.items() if d > cutoff}
        if plat == "macos":
            strain[hid] = {d: v for d, v in days.items() if d > (today - dt.timedelta(days=45)).isoformat()}

        batteries = det.get("batteries") or []
        fleet_batt = batteries[0].get("health") if batteries else None
        cycles = batteries[0].get("cycle_count") if batteries else None
        score = dh.get("battery_health_score") or None
        battery_service = score == "replace" or (fleet_batt or "").lower().startswith("service")
        mem = h.get("memory") or 0
        ram = round(mem / 2**30) if mem else None
        warranty = (det.get("vitals") or {}).get(WARRANTY_VITAL) or None
        os_version = h.get("os_version") or ""
        model = mkt or " ".join(x for x in (h.get("hardware_vendor") if plat != "macos" else "", h.get("hardware_model")) if x)
        m = {
            "id": h["id"], "name": host_name(h), "fleet": h.get("team_name") or "Unassigned", "p": plat,
            "model": model, "model_id": h.get("hardware_model") or "",
            "serial": h.get("hardware_serial") or "",
            "chip": cpu["label"] if cpu else ((h.get("cpu_brand") or "").strip() or "Unknown"),
            "chip_family": cpu["family"] if cpu else "Unknown",
            "cpu_year": cpu["year"] if cpu else None,
            "gens_behind": cpu["gens_behind"] if cpu else None,
            "old": bool(cpu and cpu["old"]),
            "year": year, "age": (today.year - year) if year else None,
            "ram": ram, "os": os_version,
            "windows10": plat == "windows" and "Windows 10" in os_version,
            "battery_pct": float(dh["battery_health_pct"]) if dh.get("battery_health_pct") else None,
            "battery_cycles": int(dh["battery_cycles"]) if (dh.get("battery_cycles") or "").isdigit() else cycles,
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
        if is_virtual(h):
            m["verdict"] = "virtual"
        elif seen_days is None or seen_days > STALE_DAYS:
            m["verdict"] = "stale"
        else:
            m["verdict"] = lifecycle_verdict(m)
        devices.append(m)

    # Hosts that are gone from Fleet stop accruing history.
    live = {str(m["id"]) for m in devices}
    for hid in list(strain):
        if hid not in live:
            del strain[hid]

    counts = {}
    for m in devices:
        counts[m["verdict"]] = counts.get(m["verdict"], 0) + 1
    summary_hist[today.isoformat()] = counts
    hist["summary"] = trim_daily(summary_hist, 400, today)
    days_collected = len({d for v in strain.values() for d in v})
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "history_since": hist["since"], "history_days": days_collected,
        "devices": devices, "current_gen": CURRENT_APPLE_GEN, "old_cpu_years": OLD_CPU_YEARS,
        "stale_days": STALE_DAYS, "warranty_vital": WARRANTY_VITAL,
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


def clean_vendor(v, name):
    """'Google LLC' -> 'Google', 'Microsoft Corporation' -> 'Microsoft'."""
    v = re.sub(r"[,.]?\s*\b(inc|incorporated|llc|ltd|limited|corp|corporation|gmbh|co|s\.a|b\.v|pty|plc|pbc)\b\.?", "", v or "", flags=re.I).strip(" ,.")
    if not v:
        return name
    return VENDOR_NAMES.get(v.lower().replace(" ", ""), v)


def tracked_apps(snap, hid, plat):
    """User-facing apps on one host: {key: (name, version, vendor, last_opened, bundle_id)}."""
    out = {}
    for name, ver, source, bid, last, path, vendor in software_rows(snap, hid):
        name = html.unescape(name)
        if plat == "macos":
            if source != "apps" or not path.startswith("/Applications/") or path.count(".app") != 1:
                continue
            if bid.startswith("com.apple."):
                continue
            key, vend = bid or name, vendor_of(bid, name)
        elif plat == "windows":
            # Fleet matches Windows prefetch data to a program by its install
            # folder, so a program without one never gets a last-opened time.
            if source != "programs" or not path or WINDOWS_BUILTIN_RE.search(name):
                continue
            if re.search(r"microsoft", vendor or "", re.I) and not MICROSOFT_SEATS_RE.search(name):
                continue
            key, vend = "win:" + name, clean_vendor(vendor, name)
        else:
            if source not in ("deb_packages", "rpm_packages") or name not in LINUX_DESKTOP_APPS:
                continue
            key = "linux:" + name
            name, vend = LINUX_DESKTOP_APPS[name]
        prev = out.get(key)
        if prev is None or vcmp(ver, prev[1]) > 0:
            out[key] = (name, ver, vend, last, bid)
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
    def has_usage_data(hid, h):
        # With Windows prefetch (SysMain) turned off, no program ever gets a
        # last-opened time, and every seat on the computer would look idle.
        if platform_of(h) != "windows":
            return True
        return any(r[2] == "programs" and r[4] for r in snap["software"].get(hid, []))

    active = [hid for hid, h in hosts.items()
              if platform_of(h) and not is_virtual(h) and is_active(h, now) and has_usage_data(hid, h)]

    apps = {}
    for hid in active:
        plat = platform_of(hosts[hid])
        for key, (name, ver, vend, last, bid) in tracked_apps(snap, hid, plat).items():
            a = apps.setdefault(key, {"key": key, "names": {}, "bid": bid, "vendor": vend, "p": plat, "hosts": []})
            a["names"][name] = a["names"].get(name, 0) + 1
            opened = parse_ts(last) if last else None
            ds = round(days_between(opened, now)) if opened else None
            a["hosts"].append([int(hid), ver, ds])

    rows = []
    for a in apps.values():
        name = max(a["names"], key=a["names"].get)
        rows.append({"key": a["key"], "name": name, "vendor": a["vendor"], "p": a["p"],
                     "kind": app_kind(name, a["bid"]), "h": sorted(a["hosts"], key=lambda x: (x[2] is not None, -(x[2] or 0)))})
    seats = [r for r in rows if r["kind"] == "app"]
    total = sum(len(r["h"]) for r in seats)
    idle = sum(1 for r in seats for x in r["h"] if x[2] is None or x[2] >= IDLE_DAYS)
    summary_hist[today.isoformat()] = {"seats": total, "idle": idle}
    hist["summary"] = trim_daily(summary_hist, 400, today)
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "hosts": host_map(hosts, active),
        "apps": sorted(rows, key=lambda r: -len(r["h"])),
        "idle_days": IDLE_DAYS, "min_installs": MIN_INSTALLS, "stale_days": STALE_DAYS,
        "trend": [{"date": d, **v} for d, v in sorted(hist["summary"].items())],
    }


# ---------------------------------------------------------------- patch velocity

def windows_os(os_version):
    """('Windows 11 25H2', '10.0.26200.9457'), or None for Insider builds."""
    if re.search(r"insider|preview", os_version, re.I):
        return None
    m = re.search(r"(Windows \d+)\b.*?(\d\dH\d)?\s*(10\.0\.\d+\.\d+)", os_version)
    if not m:
        return None
    return (f"{m.group(1)} {m.group(2)}" if m.group(2) else m.group(1)), m.group(3)


def patch_titles(snap, hid, h):
    """Version per tracked title for one host: apps plus the OS. macOS and Windows only."""
    out = {}
    plat = platform_of(h)
    if plat not in ("macos", "windows"):
        return out
    for key, (name, ver, vend, last, bid) in tracked_apps(snap, hid, plat).items():
        if app_kind(name, bid) == "support":
            continue
        out[("app:" + key) if plat == "macos" else key] = (name, ver)
    osv = h.get("os_version") or ""
    if plat == "macos" and osv:
        out["os:macOS"] = ("macOS", osv.replace("macOS ", "").strip())
    elif plat == "windows" and (w := windows_os(osv)):
        out["os:" + w[0]] = w
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
    hist.setdefault("schema", 1)
    hist.setdefault("since", D)
    versions = hist.setdefault("versions", {})
    names = hist.setdefault("names", {})
    state = hist.setdefault("hosts", {})
    events = hist.setdefault("events", [])
    summary_hist = hist.setdefault("summary", {})
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    # A title the history hasn't seen before starts as a baseline: its versions
    # weren't released today, so they aren't timed.
    known_titles = set(versions)

    betas = {(r.get("os_version") or "") for r in snap.get("os_health", {}).values()
             if re.search(r"[a-z]$", r.get("os_build") or "")}

    current = {}
    for hid, h in hosts.items():
        if is_virtual(h):
            continue
        titles = patch_titles(snap, hid, h)
        current[hid] = titles
        prev_host = state.get(hid)
        new_host = {}
        for key, (name, ver) in titles.items():
            names[key] = name
            vs = versions.setdefault(key, {})
            if ver not in vs:
                vs[ver] = D if key in known_titles else None
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
        if hid not in current:
            del state[hid]
    cutoff180 = (today - dt.timedelta(days=180)).isoformat()
    hist["events"] = events = [e for e in events if e[5] >= cutoff180]

    # Per-title installs on hosts that still check in. The page aggregates these
    # (per fleet and platform) against the fleet-wide newest release.
    active = [hid for hid in current if is_active(hosts[hid], now)]
    by_title = {}
    for hid in active:
        for key, (name, ver) in current[hid].items():
            by_title.setdefault(key, []).append([int(hid), ver])
    recent = (today - dt.timedelta(days=45)).isoformat()
    titles, index = [], {}
    for key, hv in sorted(by_title.items()):
        if len(hv) < 2:
            continue
        plat = platform_of(hosts[str(hv[0][0])])
        newest = newest_release([v for _, v in hv], betas)
        index[key] = len(titles)
        titles.append({
            "key": key, "name": names.get(key, key), "type": "os" if key.startswith("os:") else "app", "p": plat,
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
