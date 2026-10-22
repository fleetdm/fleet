#!/usr/bin/env python3
"""Build a DEX dashboard page from a collect.py snapshot.

  build.py --page lifecycle|software|patch|... --snapshot snap.json \
           --templates DIR --history in.json --out-html index.html --out-history out.json
  build.py --page overview --snapshot snap.json --templates DIR --out-html overview.html \
           --page-history lifecycle=out.json ...

The history file carries state between daily runs (strain days, version
first-seen dates). Pass a missing or empty path on the first run. The overview
keeps no history: it reads the histories the other pages just wrote.
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
TREND_DAYS = 366
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
# Linux apps whose vendors ship their own packages, so a version means the same
# thing on every distribution. The distribution's own builds (LibreOffice, VLC,
# Ubuntu's Firefox snap wrapper) update with the OS, so Patch velocity skips them.
LINUX_VENDOR_PACKAGES = {
    "google-chrome-stable", "microsoft-edge-stable", "brave-browser", "code", "cursor", "sublime-text",
    "slack-desktop", "zoom", "teams-for-linux", "1password", "docker-desktop", "spotify-client", "discord",
    "signal-desktop", "dbeaver-ce", "obsidian", "insomnia", "warp-terminal", "github-desktop", "claude-desktop",
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


def count_into(day, fleet, plat, **counts):
    """Daily trend counts are kept per fleet and platform, so the page's filters can sum them."""
    row = day.setdefault(f"{fleet}|{plat}", {})
    for k, v in counts.items():
        if v:
            row[k] = row.get(k, 0) + v


def save_trend(hist, today, day):
    """Store today's counts and return the trend as [[date, counts], ...]."""
    trend = trim_daily({**hist.get("trend", {}), today.isoformat(): day}, TREND_DAYS, today)
    hist["trend"] = trend
    return [[d, trend[d]] for d in sorted(trend)]


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

    day = {}
    for m in devices:
        scored = m["verdict"] not in ("stale", "virtual")
        count_into(day, m["fleet"], m["p"], **{m["verdict"]: 1}, scored=int(scored), old=int(scored and m["old"]))
    days_collected = len({d for v in strain.values() for d in v})
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "history_since": hist["since"], "history_days": days_collected,
        "devices": devices, "current_gen": CURRENT_APPLE_GEN, "old_cpu_years": OLD_CPU_YEARS,
        "stale_days": STALE_DAYS, "warranty_vital": WARRANTY_VITAL,
        "trend": save_trend(hist, today, day),
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
    day = {}
    for r in rows:
        if r["kind"] != "app":
            continue
        for hid, _, ds in r["h"]:
            h = hosts[str(hid)]
            count_into(day, h.get("team_name") or "Unassigned", r["p"], seats=1, idle=int(ds is None or ds >= IDLE_DAYS))
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "hosts": host_map(hosts, active),
        "apps": sorted(rows, key=lambda r: -len(r["h"])),
        "idle_days": IDLE_DAYS, "min_installs": MIN_INSTALLS, "stale_days": STALE_DAYS,
        "trend": save_trend(hist, today, day),
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


def linux_os(os_version):
    """('Ubuntu 24.04', '24.04.5'): the point release within a distribution release."""
    m = re.match(r"(.+?)\s+(\d+(?:\.\d+)*)", os_version)
    if not m:
        return None
    name, ver = m.group(1), m.group(2)
    # Ubuntu-style versions are YY.MM, so the release is the first two parts.
    parts = ver.split(".")[:2] if re.search(r"ubuntu|pop|nixos", name, re.I) else ver.split(".")[:1]
    return f"{name} {'.'.join(parts)}", ver


def patch_titles(snap, hid, h):
    """Version per tracked title for one host: apps plus the OS."""
    out = {}
    plat = platform_of(h)
    if not plat:
        return out
    for key, (name, ver, vend, last, bid) in tracked_apps(snap, hid, plat).items():
        if app_kind(name, bid) == "support":
            continue
        if plat == "linux" and key.removeprefix("linux:") not in LINUX_VENDOR_PACKAGES:
            continue
        out[("app:" + key) if plat == "macos" else key] = (name, ver)
    osv = h.get("os_version") or ""
    if plat == "macos" and osv:
        out["os:macOS"] = ("macOS", osv.replace("macOS ", "").strip())
    elif plat == "windows" and (w := windows_os(osv)):
        out["os:" + w[0]] = w
    elif plat == "linux" and (lx := linux_os(osv)):
        out["os:" + lx[0]] = lx
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

    def fleet_of(hid):
        return hosts[str(hid)].get("team_name") or "Unassigned"

    day = {}
    for t in titles:
        if len(t["i"]) < MIN_INSTALLS:
            continue
        for hid, v in t["i"]:
            current = int(vcmp(v, t["newest"]) >= 0)
            if t["type"] == "app":
                count_into(day, fleet_of(hid), t["p"], app_installs=1, app_current=current)
            else:
                count_into(day, fleet_of(hid), t["p"], os_installs=1, os_current=current)
    for _, _, _, hid, _, applied in events:
        if applied == D and str(hid) in hosts:
            count_into(day, fleet_of(hid), platform_of(hosts[str(hid)]), updates=1)
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"],
        "history_since": hist["since"], "sla_days": PATCH_SLA_DAYS, "stale_days": STALE_DAYS,
        "min_installs": MIN_INSTALLS, "hosts": host_map(hosts, active),
        "titles": titles, "events": measured, "betas": sorted(betas),
        "trend": save_trend(hist, today, day),
    }


# ---------------------------------------------------------------- AI governance

AI_WINDOWS = (7, 14, 30)
AI_APPS = {
    "claude-desktop": "Claude", "chatgpt": "ChatGPT", "ollama": "Ollama", "lm-studio": "LM Studio",
    "lm-studio-cli": "LM Studio", "jan": "Jan", "gpt4all": "GPT4All", "msty": "Msty", "anythingllm": "AnythingLLM",
    "comet": "Comet", "dia": "Dia", "perplexity": "Perplexity", "cursor": "Cursor", "windsurf": "Windsurf",
    "antigravity": "Antigravity", "trae": "Trae",
}
AI_AGENTS = {
    "claude-code": "Claude Code", "gemini-cli": "Gemini CLI", "codex": "Codex", "aider": "Aider", "goose": "Goose",
    "opencode": "OpenCode", "cline": "Cline", "continue-cli": "Continue", "cursor-agent": "Cursor",
    "amazon-q": "Amazon Q", "grok": "Grok",
}
AI_PLUGINS = [
    ("github.copilot", "GitHub Copilot"), ("anthropic.claude-code", "Claude Code"), ("openai.chatgpt", "Codex"),
    ("saoudrizwan.claude-dev", "Cline"), ("rooveterinaryinc.roo-cline", "Roo Code"), ("kilocode.", "Kilo Code"),
    ("google.geminicodeassist", "Gemini Code Assist"), ("sourcegraph.cody", "Cody"), ("codeium.", "Windsurf"),
    ("tabnine.", "Tabnine"), ("supermaven.", "Supermaven"), ("amazonwebservices.amazon-q", "Amazon Q"),
    ("visualstudioexptteam.vscodeintellicode", "IntelliCode"), ("augment.", "Augment"), ("continue.", "Continue"),
]
AI_EXTENSIONS = {"Claude": "Claude", "Gemini in Chrome": "Gemini in Chrome", "ChatGPT": "ChatGPT",
                 "ChatGPT search": "ChatGPT", "Edge Copilot Bridge": "Microsoft Copilot"}
# Plugin display names that vary by editor (JetBrains vs VS Code) or edition.
AI_PLUGIN_NAMES = [
    (re.compile(r"^GitHub Copilot(?! for Azure)"), "GitHub Copilot"), (re.compile(r"^Claude Code"), "Claude Code"),
    (re.compile(r"^Codex\b"), "Codex"), (re.compile(r"\bMCP\b"), "MCP server extensions"),
]
# Live sockets name the process, not the product. Claude Code's native binary
# is named after its version (~/.local/share/claude/versions/2.1.300).
AI_SOCKETS = [
    (re.compile(r"claude/versions/|(^|[/\\])claude(\.exe)?$", re.I), "Claude Code"),
    (re.compile(r"Claude\.app|[/\\]Claude[/\\]|claude-desktop|^Claude( Helper.*)?$", re.I), "Claude"),
    (re.compile(r"copilot-language-server", re.I), "GitHub Copilot"),
    (re.compile(r"ollama", re.I), "Ollama"),
    (re.compile(r"lm.?studio|(^|[/\\])lms$", re.I), "LM Studio"),
    (re.compile(r"perplexity", re.I), "Perplexity"),
    (re.compile(r"(^|[/\\])codex(\.exe)?$|Codex\.app", re.I), "Codex"),
    (re.compile(r"Cursor\.app|(^|[/\\])cursor(-agent)?$|^Cursor( Helper.*)?$", re.I), "Cursor"),
    (re.compile(r"ChatGPT", re.I), "ChatGPT"),
    (re.compile(r"windsurf", re.I), "Windsurf"),
    (re.compile(r"gemini", re.I), "Gemini CLI"),
]
AI_SERVICES = {"ollama": "Ollama", "lm-studio": "LM Studio", "jan": "Jan"}
# Software inventory names whose last-opened time counts as using the product.
AI_INVENTORY_APPS = {
    "Claude": "Claude", "ChatGPT": "ChatGPT", "ChatGPT Atlas": "ChatGPT", "ChatGPT Classic": "ChatGPT",
    "Cursor": "Cursor", "Windsurf": "Windsurf", "LM Studio": "LM Studio", "Ollama": "Ollama",
    "Perplexity": "Perplexity", "Comet": "Comet", "Antigravity": "Antigravity", "Jan": "Jan", "Msty": "Msty",
    "GPT4All": "GPT4All", "AnythingLLM": "AnythingLLM", "Dia": "Dia", "Trae": "Trae",
}
AI_EDITORS = {"vscode": "VS Code", "vscode-insiders": "VS Code", "vscode-server": "VS Code", "code-server": "VS Code",
              "vscodium": "VSCodium", "cursor": "Cursor", "windsurf": "Windsurf", "trae": "Trae",
              "antigravity": "Antigravity", "antigravity-ide": "Antigravity", "zed": "Zed", "sublime": "Sublime Text",
              "neovim": "Neovim", "vim": "Vim", "emacs": "Emacs"}
AI_BROWSERS = {"chrome": "Chrome", "chrome-beta": "Chrome", "edge": "Edge", "brave": "Brave", "arc": "Arc",
               "opera": "Opera", "vivaldi": "Vivaldi", "chromium": "Chromium", "comet": "Comet", "dia": "Dia",
               "firefox": "Firefox", "zen": "Zen", "librewolf": "LibreWolf", "waterfox": "Waterfox"}
AI_MCP_CLIENTS = {"claude-desktop": "Claude", "claude-code": "Claude Code", "cursor": "Cursor", "windsurf": "Windsurf",
                  "vscode": "VS Code", "zed": "Zed", "cline": "Cline", "roo": "Roo Code", "continue": "Continue",
                  "jan": "Jan", "gemini": "Gemini CLI", "hermes": "Hermes", "openclaw": "OpenClaw", "grok": "Grok",
                  "codex": "Codex", "process": "Running, no config"}
AI_INSTRUCTION_TOOLS = {"claude": "CLAUDE.md", "codex": "AGENTS.md", "gemini": "GEMINI.md", "opencode": "OpenCode rules",
                        "cursor": "Cursor rules", "windsurf": "Windsurf rules", "cline": "Cline rules", "roo": "Roo rules",
                        "copilot": "Copilot instructions"}
# Row types that mean a product is installed, as opposed to configured or seen.
AI_INSTALL_TYPES = {"apps", "agents", "ide_plugins", "browser_extension"}
HOME_RE = re.compile(r"^(/Users/[^/]+|/home/[^/]+|/root|[A-Za-z]:\\Users\\[^\\]+)", re.I)


def ai_slug(name):
    return re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-")


def short_path(p):
    """Home-relative path, so usernames in paths aren't published."""
    return HOME_RE.sub("~", p or "")


def ai_detail(t):
    try:
        d = json.loads(t.get("detail") or "{}")
        return d if isinstance(d, dict) else {}
    except ValueError:
        return {}


def ai_product(t):
    typ, ident, name = t.get("type"), t.get("identifier") or "", t.get("name") or ""
    if typ == "apps":
        return AI_APPS.get(ident) or name or None
    if typ == "agents":
        return AI_AGENTS.get(name) or AI_AGENTS.get(ident) or None
    if typ == "ide_plugins":
        low = ident.lower()
        for prefix, product in AI_PLUGINS:
            if low.startswith(prefix):
                return product
        for rx, product in AI_PLUGIN_NAMES:
            if rx.search(name):
                return product
        return name if name and not name.startswith("%") else ident or None
    if typ == "browser_extension":
        return AI_EXTENSIONS.get(name, name) or None
    if typ == "sockets":
        if ident in AI_SERVICES:
            return AI_SERVICES[ident]
        for rx, product in AI_SOCKETS:
            if rx.search(t.get("path") or "") or rx.search(name):
                return product
    return None


def ai_surface(t, det):
    typ = t.get("type")
    if typ == "apps":
        return "Desktop app"
    if typ == "agents":
        return "Terminal"
    if typ == "ide_plugins":
        src = t.get("source") or ""
        return AI_EDITORS.get(src) or ("JetBrains IDEs" if det.get("editor_family") == "jetbrains" else src.title() or "Editor")
    if typ == "browser_extension":
        return AI_BROWSERS.get(t.get("source") or "", (t.get("source") or "Browser").title()) + " extension"
    return None


def mcp_name(t, det):
    name = (t.get("name") or t.get("identifier") or "").strip()
    if not name or len(name) > 60 or not re.fullmatch(r"[\w .@/:+=~-]+", name):
        cmd = os.path.basename((det.get("command") or "").replace("\\", "/")) or "unnamed"
        return f"{cmd} (inline script)"
    return name


def build_ai(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    D = today.isoformat()
    hist.setdefault("schema", 1)
    hist.setdefault("since", D)
    live = hist.setdefault("live", {})
    first_seen = hist.setdefault("first_seen", {})
    # A tool that appears on the first run is the baseline, not a new tool.
    baseline = not first_seen
    ai = snap.get("ai_tools") or {}
    reports = ai.get("hosts") or {}
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    ids = [hid for hid, h in hosts.items() if platform_of(h) and is_active(h, now)]

    products, mcp, instructions, findings, matchers = {}, {}, {}, [], {}

    def product_host(name, hid):
        p = products.setdefault(name, {"name": name, "h": {}})
        return p["h"].setdefault(hid, {"s": set(), "v": "", "live": False, "opened": None})

    for hid in ids:
        h = hosts[hid]
        plat = platform_of(h)
        # Last-opened times from Fleet's software inventory (apps only).
        for sname, ver, source, bid, last, path, vendor in software_rows(snap, hid):
            prod = AI_INVENTORY_APPS.get(html.unescape(sname)) if source in ("apps", "programs") else None
            if prod and last and (opened := parse_ts(last)):
                e = product_host(prod, hid)
                d = days_between(opened, now)
                e["opened"] = d if e["opened"] is None else min(e["opened"], d)

        rep = reports.get(hid)
        if not rep:
            continue
        fetched = parse_ts(rep.get("last_fetched"))
        fresh = bool(fetched and days_between(fetched, now) <= 1.5)
        for t in rep.get("tools") or []:
            typ = t.get("type")
            det = ai_detail(t)
            flags = [f for f in (t.get("risk_flags") or "").split(",") if f]
            prod = ai_product(t)
            running = str(t.get("running") or "") == "1" or typ == "sockets"
            if prod:
                e = product_host(prod, hid)
                if (surface := ai_surface(t, det)):
                    e["s"].add(surface)
                if typ in AI_INSTALL_TYPES and t.get("version") and not e["v"]:
                    e["v"] = t["version"]
                if running and fresh:
                    e["live"] = True
                if typ in AI_INSTALL_TYPES:
                    field = "name" if typ == "agents" else "identifier"
                    if t.get(field):
                        matchers.setdefault(prod, set()).add((typ, field, t[field]))
            item = prod or ""
            if typ == "mcp_server":
                name = mcp_name(t, det)
                item = name
                m = mcp.setdefault(name, {"name": name, "transports": set(), "remote": set(), "commands": set(),
                                          "caps": set(), "h": {}})
                m["transports"].add(det.get("transport") or ("http" if t.get("location") == "remote" else "stdio"))
                if t.get("endpoint"):
                    from urllib.parse import urlparse
                    host = urlparse(t["endpoint"]).hostname
                    if host:
                        m["remote"].add(host)
                if det.get("command"):
                    m["commands"].add(os.path.basename(det["command"].replace("\\", "/")))
                m["caps"].update(c for c in (det.get("capabilities") or "").split(",") if c)
                client = AI_MCP_CLIENTS.get(t.get("source") or "", t.get("source") or "")
                eh = m["h"].setdefault(hid, {"clients": set(), "running": False, "flags": set()})
                eh["clients"].add(client)
                eh["running"] = eh["running"] or (det.get("source_type") in ("process", "both") and fresh)
                eh["flags"].update(flags)
            elif typ == "agent_instruction":
                tool = AI_INSTRUCTION_TOOLS.get(t.get("source") or "", t.get("source") or "Other")
                instructions.setdefault(tool, {}).setdefault(hid, 0)
                instructions[tool][hid] += 1
                item = short_path(t.get("path")) or t.get("name") or ""
            elif typ == "sockets":
                item = prod or t.get("name") or ""
            for flag in flags:
                where = short_path(t.get("path")) if typ == "mcp_server" else ""
                findings.append([int(hid), flag, typ, item, where])

    # Live signals carry over day to day, so "active in the last 30 days" holds.
    cutoff = (today - dt.timedelta(days=35)).isoformat()
    for hid in list(live):
        if hid not in hosts:
            del live[hid]
    for name, p in products.items():
        for hid, e in p["h"].items():
            if e["live"]:
                live.setdefault(hid, {})[name] = D
    for hid in list(live):
        live[hid] = {k: v for k, v in live[hid].items() if v >= cutoff}
        if not live[hid]:
            del live[hid]

    def days_ago(name, hid, e):
        last = (live.get(hid) or {}).get(name)
        d_live = (today - dt.date.fromisoformat(last)).days if last else None
        opts = [x for x in (d_live, e["opened"]) if x is not None]
        return round(min(opts), 1) if opts else None

    out_products = []
    for name, p in sorted(products.items(), key=lambda kv: -len(kv[1]["h"])):
        key = "p:" + ai_slug(name)
        if key not in first_seen:
            first_seen[key] = None if baseline else D
        rows = []
        for hid, e in p["h"].items():
            rows.append([int(hid), sorted(e["s"]), days_ago(name, hid, e), e["v"]])
        out_products.append({"k": ai_slug(name), "name": name, "h": rows, "new": first_seen[key],
                             "m": sorted(matchers.get(name, set()))[:40]})

    out_mcp = []
    for name, m in sorted(mcp.items(), key=lambda kv: -len(kv[1]["h"])):
        key = "m:" + name
        if key not in first_seen:
            first_seen[key] = None if baseline else D
        out_mcp.append({"name": name, "transports": sorted(m["transports"]), "remote": sorted(m["remote"])[:5],
                        "commands": sorted(m["commands"])[:5], "caps": sorted(m["caps"]), "new": first_seen[key],
                        "h": [[int(hid), sorted(e["clients"]), int(e["running"]), sorted(e["flags"])]
                              for hid, e in m["h"].items()]})

    reported = {hid for hid in ids if hid in reports}
    day = {}
    sev = {"high": 0, "medium": 0}
    flag_sev = {f: s for f, (s, _, _) in AI_RISKS.items()}
    by_host_sev = {}
    for hid, flag, *_ in findings:
        s = flag_sev.get(flag, "low")
        by_host_sev.setdefault(str(hid), {"high": 0, "medium": 0, "low": 0})[s] += 1
    for hid in ids:
        h = hosts[hid]
        f, plat = h.get("team_name") or "Unassigned", platform_of(h)
        mine = [(p, p["h"][hid]) for p in products.values() if hid in p["h"]]
        active_today = [p for p, e in mine if (d := days_ago(p["name"], hid, e)) is not None and d < 1]
        hs = by_host_sev.get(hid, {})
        count_into(day, f, plat, reported=int(hid in reported), ai=int(bool(mine)), act=int(bool(active_today)),
                   high=hs.get("high", 0), med=hs.get("medium", 0),
                   **{"a:" + ai_slug(p["name"]): 1 for p in active_today},
                   **{"i:" + ai_slug(p["name"]): 1 for p, _ in mine})

    trimmed = dict(sorted(first_seen.items()))
    hist["first_seen"] = trimmed
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"], "history_since": hist["since"],
        "windows": AI_WINDOWS, "stale_days": STALE_DAYS,
        "source": {"report": ai.get("report"), "clipped": bool(ai.get("clipped")), "reported": len(reported),
                   "per_host": ai.get("per_host", 0)},
        "hosts": {hid: {**host_map(hosts, [hid])[hid], "r": int(hid in reported)} for hid in ids},
        "products": out_products, "mcp": out_mcp, "findings": findings,
        "instructions": [{"tool": tool, "h": [[int(hid), n] for hid, n in hs.items()]}
                         for tool, hs in sorted(instructions.items(), key=lambda kv: -len(kv[1]))],
        "risks": {f: {"sev": s, "label": label, "about": about} for f, (s, label, about) in AI_RISKS.items()},
        "trend": save_trend(hist, today, day),
    }


AI_RISKS = {
    "skip_permissions_runtime": ("high", "Agent running with permission checks off",
                                 "An AI agent is running with a flag that approves every action on its own, such as --dangerously-skip-permissions, --yolo, or --full-auto. It can run commands and change files without asking."),
    "bypass_permissions": ("high", "Claude Code set to bypass permissions",
                           "Claude Code's settings make bypassPermissions the default, so every session skips approval prompts."),
    "plaintext_secret": ("high", "Credential stored in an MCP config",
                         "An MCP server's config sets an environment variable or header named like a credential (token, API key, password, authorization). Its value sits in the config file in plain text."),
    "injection_markers": ("high", "Instruction file with injection phrases",
                          "An agent instruction file contains phrases common in prompt-injection attacks, such as \"ignore previous instructions\", piping a download to a shell, or reading SSH keys."),
    "hidden_unicode": ("high", "Instruction file with hidden characters",
                       "An agent instruction file contains invisible Unicode characters, which can hide instructions from the people who review it."),
    "mcp_shell_exec": ("high", "MCP server that runs shell commands",
                       "The MCP server's name or launch command matches a server that can run shell commands on the computer."),
    "remote_fetch_exec": ("medium", "MCP server downloaded at every launch",
                          "The MCP server starts through a runner such as npx or uvx, which downloads code from a package registry and runs it each time."),
    "unpinned_dependency": ("medium", "MCP server package not pinned",
                            "The MCP server's package has no fixed version, or uses latest, so each new release runs without review."),
    "cleartext_endpoint": ("medium", "Remote MCP server over plain HTTP",
                           "The MCP server is reached over http://, so its traffic, including any credentials, isn't encrypted."),
    "auto_accept_edits": ("medium", "Claude Code set to accept edits",
                          "Claude Code's settings make acceptEdits the default, so its file changes apply without review."),
    "mcp_fs_write": ("medium", "MCP server that writes files",
                     "The MCP server's name or launch command matches a server that can write files on the computer."),
    "world_writable": ("medium", "Instruction file anyone can edit",
                       "Other user accounts on the computer can change this agent instruction file, and with it what the agent is told to do."),
    "sideloaded_unverified": ("medium", "Browser extension from outside the store",
                              "The extension wasn't installed from the browser's store, or isn't signed, so no store review covers it."),
    "broad_host_permissions": ("medium", "Browser extension that can read every site",
                               "The extension asks for access to all websites, so it can read and change pages you visit."),
    "world_readable_config": ("low", "MCP config other users can read",
                              "Other user accounts on the computer can read this MCP config file, including any credentials in it."),
}

# ---------------------------------------------------------------- connectivity

WIFI_WEAK = {"fair", "poor", "unusable"}
WEAK_DAYS = 3


def num(v):
    try:
        return float(v)
    except (TypeError, ValueError):
        return None


def fresh_row(row, now, days=1.5):
    t = parse_ts((row or {}).get("last_fetched"))
    return bool(t and days_between(t, now) <= days)


def build_connectivity(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    D = today.isoformat()
    hist.setdefault("schema", 1)
    hist.setdefault("since", D)
    wifi_hist = hist.setdefault("wifi", {})
    dex = snap.get("dex") or {}
    wifi, vpn = dex.get("wifi") or {}, dex.get("vpn") or {}
    info = snap.get("report_info") or {}
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    week_ago = (today - dt.timedelta(days=7)).isoformat()

    rows, day = [], {}
    for hid, h in hosts.items():
        plat = platform_of(h)
        if not plat:
            continue
        seen = parse_ts(h.get("seen_time"))
        age = days_between(seen, now) if seen else None
        w = wifi.get(hid) if plat == "macos" else None
        v = vpn.get(hid) if plat == "macos" else None
        rssi = num((w or {}).get("rssi"))
        # macOS reports 0 dBm when location access for Wi-Fi details is withheld.
        on_wifi = bool(w and fresh_row(w, now) and rssi not in (None, 0))
        quality = w.get("signal_quality") if on_wifi else None
        if on_wifi:
            wifi_hist.setdefault(hid, {})[D] = quality
        recent = {d: q for d, q in (wifi_hist.get(hid) or {}).items() if d > week_ago}
        path = v.get("network_confidence") if v and fresh_row(v, now) else None
        r = {
            "id": int(hid), "n": host_name(h), "f": h.get("team_name") or "Unassigned", "p": plat,
            "virtual": is_virtual(h), "age": round(age, 3) if age is not None else None,
            "rssi": rssi if on_wifi else None, "snr": num(w.get("snr_db")) if on_wifi else None,
            "rate": num(w.get("transmit_rate")) if on_wifi else None,
            "band": w.get("wifi_band") if on_wifi else None, "quality": quality,
            "score": w.get("dex_wifi_score") if on_wifi else None,
            "weak_days": sum(1 for q in recent.values() if q in WIFI_WEAK), "wifi_days": len(recent),
            "path": path, "tunnels": int(num(v.get("vpn_tunnels_active")) or 0) if path else None,
        }
        rows.append(r)
        count_into(day, r["f"], plat, computers=1, fresh1h=int(age is not None and age <= 1 / 24),
                   fresh24=int(age is not None and age <= 1), stale7=int(age is None or age > 7),
                   wifi=int(on_wifi), weak=int(quality in WIFI_WEAK), persistent=int(r["weak_days"] >= WEAK_DAYS),
                   tunnel=int(path == "tunnel_active"), direct=int(path == "direct_connected"),
                   offline=int(path == "disconnected"))

    cutoff = (today - dt.timedelta(days=30)).isoformat()
    for hid in list(wifi_hist):
        kept = {d: q for d, q in wifi_hist[hid].items() if d >= cutoff}
        if hid in hosts and kept:
            wifi_hist[hid] = kept
        else:
            del wifi_hist[hid]
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"], "history_since": hist["since"],
        "stale_days": STALE_DAYS, "weak_days": WEAK_DAYS, "rows": rows,
        "reports": {k: bool((info.get(k) or {}).get("found")) for k in ("wifi", "vpn")},
        "trend": save_trend(hist, today, day),
    }


# ---------------------------------------------------------------- analytics

# (key, label, domain, platforms). Platforms None means every computer.
ANALYTICS_SOURCES = [
    ("checkin", "Fleet check-in", "Fleet agent", None),
    ("details", "Host details", "Hardware and health", None),
    ("software", "Software inventory", "Software", None),
    ("policies", "Policies", "Security", None),
    ("device_health", "Device health report", "Hardware and health", {"macos"}),
    ("os_health", "OS health report", "Hardware and health", {"macos"}),
    ("ai_tools", "AI tools report", "Software", None),
    ("wifi", "Wi-Fi report", "Network", {"macos"}),
    ("vpn", "VPN gate report", "Network", {"macos"}),
    ("mac_posture", "macOS security posture report", "Security", {"macos"}),
    ("win_security", "Windows Security Center report", "Security", {"windows"}),
    ("win_secure_boot", "Secure Boot report", "Security", {"windows"}),
    ("win_tpm", "TPM report", "Security", {"windows"}),
]
HOST_TIMESTAMPS = {"checkin": "seen_time", "details": "detail_updated_at", "software": "software_updated_at",
                   "policies": "policy_updated_at"}
FRESH_DAYS = 7
UPTIME_OVERDUE_DAYS = 30


def posture(plat, h, dex, hid):
    """Security controls for one computer: True, False, or None when nothing reported."""
    def flag(v):
        return None if v is None else str(v).strip() not in ("", "0", "None")
    c = {"disk": h.get("disk_encryption_enabled")}
    if plat == "macos":
        r = (dex.get("mac_posture") or {}).get(hid)
        for k in ("firewall", "gatekeeper", "sip"):
            # The report selects 1 or nothing, so a reported row with no value means off.
            c[k] = (flag(r.get(k)) or False) if r else None
    elif plat == "windows":
        sc = (dex.get("win_security") or {}).get(hid)
        sb = (dex.get("win_secure_boot") or {}).get(hid)
        tpm = (dex.get("win_tpm") or {}).get(hid)
        c["firewall"] = (sc.get("firewall") == "Good") if sc else None
        c["antivirus"] = (sc.get("antivirus") == "Good") if sc else None
        c["secure_boot"] = flag(sb.get("secure_boot_enabled")) if sb else None
        c["tpm"] = (tpm.get("tpm_status") == "active") if tpm else None
    return {k: (None if v is None else int(bool(v))) for k, v in c.items()}


def build_analytics(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    hist.setdefault("schema", 1)
    hist.setdefault("since", today.isoformat())
    dex = dict(snap.get("dex") or {})
    dex["device_health"], dex["os_health"] = snap.get("device_health") or {}, snap.get("os_health") or {}
    ai = (snap.get("ai_tools") or {}).get("hosts") or {}
    info = dict(snap.get("report_info") or {})
    info["ai_tools"] = {"found": bool((snap.get("ai_tools") or {}).get("report")),
                        "clipped": bool((snap.get("ai_tools") or {}).get("clipped"))}
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    ids = [hid for hid, h in hosts.items() if platform_of(h) and is_active(h, now)]

    def found(key):
        """Fleet's own timestamps always exist; a report counts only once someone adds it."""
        return key in HOST_TIMESTAMPS or bool((info.get(key) or {}).get("found"))

    vers = {"orbit": [], "osquery": [], "desktop": []}
    for hid in ids:
        h = hosts[hid]
        for k, field in (("orbit", "orbit_version"), ("osquery", "osquery_version"), ("desktop", "fleet_desktop_version")):
            if h.get(field):
                vers[k].append(h[field])
    newest = {k: newest_release(v, set()) if v else None for k, v in vers.items()}

    out, day = {}, {}
    for hid in ids:
        h = hosts[hid]
        plat = platform_of(h)
        ages = {}
        for key, _, _, plats in ANALYTICS_SOURCES:
            if plats and plat not in plats:
                continue
            if key in HOST_TIMESTAMPS:
                t = parse_ts(h.get(HOST_TIMESTAMPS[key]))
            elif key == "ai_tools":
                t = parse_ts((ai.get(hid) or {}).get("last_fetched"))
            else:
                t = parse_ts(((dex.get(key) or {}).get(hid) or {}).get("last_fetched"))
            ages[key] = round(days_between(t, now), 2) if t else None
        controls = posture(plat, h, dex, hid)
        v = [h.get("orbit_version") or "", h.get("osquery_version") or "", h.get("fleet_desktop_version") or ""]
        current = bool(newest["orbit"] and v[0] and vcmp(v[0], newest["orbit"]) >= 0)
        up = round((h.get("uptime") or 0) / 1e9 / 86400, 1) if h.get("uptime") else None
        out[hid] = {"n": host_name(h), "f": h.get("team_name") or "Unassigned", "p": plat, "virtual": is_virtual(h),
                    "s": ages, "c": controls, "v": v, "up": up, "fp": h.get("failing_policies"), "mdm": h.get("mdm_status")}
        all_fresh = all(a is not None and a <= FRESH_DAYS for k, a in ages.items() if found(k))
        count_into(day, out[hid]["f"], plat, computers=1, all_fresh=int(all_fresh),
                   disk_known=int(controls["disk"] is not None), disk_on=int(controls["disk"] == 1),
                   agent_current=int(current), failing=int((h.get("failing_policies") or 0) > 0))
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"], "history_since": hist["since"],
        "stale_days": STALE_DAYS, "fresh_days": FRESH_DAYS, "uptime_overdue": UPTIME_OVERDUE_DAYS,
        "sources": [{"key": k, "label": label, "domain": domain, "platforms": sorted(plats) if plats else None,
                     "found": found(k), "clipped": bool((info.get(k) or {}).get("clipped"))}
                    for k, label, domain, plats in ANALYTICS_SOURCES],
        "hosts": out, "newest": newest, "trend": save_trend(hist, today, day),
    }

# ---------------------------------------------------------------- sustainability

# Grams of CO2 per kWh of electricity generated, latest year per country (2025
# for most). Source: Ember, via Our World in Data's "Carbon intensity of
# electricity generation" (CC BY 4.0). Keyed by the ISO country code Fleet's
# GeoIP lookup returns.
GRID_INTENSITY = {
    "AE": 468, "AF": 131, "AG": 595, "AL": 25, "AM": 212, "AO": 185, "AR": 346, "AS": 611, "AT": 117, "AU": 525,
    "AW": 550, "AZ": 632, "BA": 571, "BB": 595, "BD": 696, "BE": 150, "BF": 562, "BG": 276, "BH": 902, "BI": 184,
    "BJ": 584, "BM": 639, "BN": 892, "BO": 481, "BR": 110, "BS": 653, "BT": 24, "BW": 851, "BY": 309, "BZ": 170,
    "CA": 191, "CD": 28, "CG": 716, "CH": 39, "CI": 405, "CK": 250, "CL": 289, "CM": 226, "CN": 525, "CO": 187,
    "CR": 24, "CU": 643, "CV": 462, "CY": 489, "CZ": 401, "DE": 330, "DJ": 450, "DK": 114, "DM": 600, "DO": 537,
    "DZ": 633, "EC": 159, "EE": 319, "EG": 563, "EH": 667, "ER": 578, "ES": 154, "ET": 23, "FI": 57, "FJ": 278,
    "FK": 1000, "FO": 347, "FR": 41, "GA": 523, "GB": 217, "GD": 667, "GE": 146, "GF": 245, "GH": 469, "GI": 591,
    "GL": 150, "GM": 667, "GN": 181, "GP": 497, "GQ": 644, "GR": 315, "GT": 301, "GU": 608, "GW": 625, "GY": 645,
    "HK": 675, "HN": 322, "HR": 159, "HT": 535, "HU": 163, "ID": 680, "IE": 257, "IL": 493, "IN": 670, "IQ": 683,
    "IR": 660, "IS": 28, "IT": 285, "JM": 563, "JO": 530, "JP": 477, "KE": 95, "KG": 153, "KH": 499, "KI": 500,
    "KM": 643, "KN": 609, "KP": 341, "KR": 417, "KW": 635, "KY": 634, "KZ": 805, "LA": 232, "LB": 390, "LC": 650,
    "LK": 329, "LR": 316, "LS": 21, "LT": 138, "LU": 123, "LV": 139, "LY": 827, "MA": 596, "MD": 633, "ME": 264,
    "MG": 432, "MK": 441, "ML": 539, "MM": 503, "MN": 816, "MO": 474, "MQ": 530, "MR": 512, "MS": 1000, "MT": 484,
    "MU": 642, "MV": 612, "MW": 55, "MX": 474, "MY": 602, "MZ": 129, "NA": 49, "NC": 561, "NE": 674, "NG": 456,
    "NI": 301, "NL": 254, "NO": 28, "NP": 24, "NR": 600, "NZ": 93, "OM": 544, "PA": 221, "PE": 238, "PF": 431,
    "PG": 514, "PH": 588, "PK": 347, "PL": 589, "PM": 600, "PR": 655, "PS": 414, "PT": 128, "PY": 25, "QA": 582,
    "RE": 394, "RO": 251, "RS": 696, "RU": 450, "RW": 354, "SA": 692, "SB": 636, "SC": 556, "SD": 154, "SE": 35,
    "SG": 497, "SH": 1000, "SI": 183, "SK": 95, "SL": 48, "SN": 540, "SO": 512, "SR": 322, "SS": 643, "ST": 556,
    "SV": 139, "SY": 706, "SZ": 131, "TC": 630, "TD": 622, "TG": 423, "TH": 546, "TJ": 73, "TL": 667, "TM": 1306,
    "TN": 560, "TO": 571, "TR": 475, "TT": 682, "TW": 633, "TZ": 345, "UA": 250, "UG": 59, "US": 384, "UY": 80,
    "UZ": 1000, "VC": 600, "VE": 86, "VG": 647, "VI": 632, "VN": 461, "VU": 500, "WS": 375, "XK": 901, "YE": 592,
    "ZA": 699, "ZM": 120, "ZW": 384
}
GRID_SOURCE = "Ember via Our World in Data (CC BY 4.0), latest year per country"
# Used for computers with no GeoIP country. Unset, it falls back to the most
# common country in the fleet.
DEFAULT_GRID = os.environ.get("DEX_GRID_INTENSITY", "")
HOST_KWH_DAYS = 35
SPOT_DAYS = 30
NEVER_SLEPT_DAYS = 7
LONG_DISPLAY_MIN = 30
# A reading this far above any Mac's draw means a bad counter, not real use.
MAX_PLAUSIBLE_WATTS = 1000
# Processes in pmset's "sleep prevented by" list that are macOS itself.
MAC_SYSTEM_BLOCKERS = {
    "powerd", "dasd", "coreaudiod", "bluetoothd", "sharingd", "useractivityd", "backupd", "mds", "mds_stores",
    "WindowServer", "runningboardd", "apsd", "cloudd", "nsurlsessiond", "softwareupdated", "locationd", "airportd",
    "mDNSResponder", "configd", "UserEventAgent", "symptomsd", "rapportd", "ControlCenter", "audiomxd",
    "com.apple.audio.coreaudiod", "kernel_task", "launchd", "AirPlayXPCHelper", "screensharingd",
}


def as_int(v):
    try:
        return int(float(str(v).strip()))
    except (TypeError, ValueError):
        return None


def prevented_by(text):
    """Process names from pmset's "1 (sleep prevented by a, b)"."""
    m = re.search(r"prevented by ([^)]*)\)", text or "")
    return sorted({p.strip() for p in m.group(1).split(",") if p.strip()}) if m else []


def gnome_value(v):
    """dconf prints GVariant text: "'nothing'\\n" or "3600\\n". Empty means the default."""
    v = (v or "").strip()
    return v.strip("'") if v else None


def split_days(start, end, kwh, hours):
    """Spread kWh and hours evenly over the UTC days between two readings."""
    out = {}
    if end <= start:
        return {end.date().isoformat(): (kwh, hours)}
    total = (end - start).total_seconds()
    t = start
    while t < end:
        nxt = min(end, dt.datetime.combine(t.date() + dt.timedelta(days=1), dt.time(), tzinfo=dt.timezone.utc))
        share = (nxt - t).total_seconds() / total
        out[t.date().isoformat()] = (kwh * share, hours * share)
        t = nxt
    return out


def add_spot(hist, hid, t, watts, today):
    """Keep one running average per host per day for single power readings."""
    if watts is None or watts <= 0 or watts > MAX_PLAUSIBLE_WATTS:
        return
    seen = hist.setdefault("spot_seen", {})
    stamp = t.isoformat() if t else today.isoformat()
    if seen.get(hid) == stamp:
        return
    seen[hid] = stamp
    day = hist.setdefault("spot", {}).setdefault(hid, {}).setdefault(today.isoformat(), [0.0, 0])
    day[0] = round(day[0] + watts, 3)
    day[1] += 1


def energy_reading(hist, hid, row):
    """Turn a new Mac energy counter reading into kWh per UTC day.

    The counter is millijoules since boot. Time comes from the host's own
    uptime, because Fleet can receive a result long after the host recorded it.
    """
    e, up, t = as_int(row.get("energy_mj")), as_int(row.get("uptime_s")), parse_ts(row.get("last_fetched"))
    if e is None or up is None or up <= 0 or t is None:
        return {}
    store = hist.setdefault("energy", {})
    last = store.get(hid)
    if last and t.timestamp() <= last["t"]:
        return {}
    store[hid] = {"e": e, "up": up, "t": t.timestamp()}
    if not last:
        return {}
    if up >= last["up"] and e >= last["e"]:
        joules, secs = (e - last["e"]) / 1000, up - last["up"]
    else:
        # Rebooted since the last reading. Count the energy since boot, minus
        # any stretch the last reading already covered.
        joules, secs = e / 1000, up
        overlap = last["t"] - (t.timestamp() - up)
        if overlap > 600:
            joules *= max(0.0, 1 - overlap / up)
            secs = max(0, up - overlap)
    if secs <= 0 or joules < 0 or joules / secs > MAX_PLAUSIBLE_WATTS:
        return {}
    return split_days(t - dt.timedelta(seconds=secs), t, joules / 3.6e6, secs / 3600)


def battery_status(det, dh):
    batts = det.get("batteries") or []
    pct = num((dh or {}).get("battery_health_pct"))
    cycles = as_int((dh or {}).get("battery_cycles"))
    if cycles is None and batts:
        cycles = batts[0].get("cycle_count")
    if not batts and pct is None:
        return None
    health = (batts[0].get("health") or "") if batts else ""
    if health.lower().startswith("service") or (pct is not None and pct < 80) or (cycles or 0) >= 1000:
        status = "service"
    elif pct is not None and pct < 90:
        status = "worn"
    else:
        status = "normal"
    return {"status": status, "pct": pct, "cycles": cycles}


def build_sustainability(snap, hist):
    now = parse_ts(snap["generated_at"])
    today = now.date()
    hist.setdefault("schema", 1)
    hist.setdefault("since", today.isoformat())
    dex = snap.get("dex") or {}
    details = snap.get("host_details") or {}
    info = snap.get("report_info") or {}
    hosts = {str(h["id"]): h for h in snap["hosts"]}
    ids = [hid for hid, h in hosts.items() if platform_of(h) and not is_virtual(h)]

    countries = [(details.get(hid) or {}).get("country") for hid in ids]
    known = [c for c in countries if c in GRID_INTENSITY]
    if num(DEFAULT_GRID):
        fallback, fallback_from = num(DEFAULT_GRID), "DEX_GRID_INTENSITY"
    elif known:
        common = max(set(known), key=known.count)
        fallback, fallback_from = GRID_INTENSITY[common], common
    else:
        fallback, fallback_from = None, None

    host_kwh = hist.setdefault("host_kwh", {})
    kwh_day = hist.setdefault("kwh_day", {})
    rows, day = [], {}
    for hid in ids:
        h, det = hosts[hid], details.get(hid) or {}
        plat, fleet_name = platform_of(h), h.get("team_name") or "Unassigned"
        key = f"{fleet_name}|{plat}"
        country = det.get("country") or ""
        grid = GRID_INTENSITY.get(country) or fallback
        up_days = round(h["uptime"] / 1e9 / 86400, 1) if h.get("uptime") else None
        r = {"id": int(hid), "n": host_name(h), "f": fleet_name, "p": plat,
             "model": det.get("hardware_marketing_name") or h.get("hardware_model") or "Unknown",
             "country": country, "grid": grid, "grid_known": country in GRID_INTENSITY, "up": up_days,
             "kind": None, "measured": False, "avg_boot_w": None, "now_w": None, "spot_w": None, "spot_n": 0,
             "sleep_off": None, "display_long": None, "never_slept": None, "fast_startup": None, "womp": None,
             "sleep_ac": None, "display_ac": None, "apps": [], "system": [],
             "battery": battery_status(det, (snap.get("device_health") or {}).get(hid))}

        if plat == "macos":
            er, sr = (dex.get("mac_energy") or {}).get(hid), (dex.get("mac_sleep") or {}).get(hid)
            laptop = "macbook" in r["model"].lower() or bool(er and er.get("power_source"))
            r["kind"] = "laptop" if laptop else "desktop"
            if er:
                r["now_w"] = num(er.get("system_watts"))
                e, up = as_int(er.get("energy_mj")), as_int(er.get("uptime_s"))
                if e is not None and up:
                    r["measured"] = True
                    r["avg_boot_w"] = round(e / 1000 / up, 2)
                    for d, (kwh, hours) in energy_reading(hist, hid, er).items():
                        slot = host_kwh.setdefault(hid, {}).setdefault(d, [0.0, 0.0])
                        slot[0], slot[1] = round(slot[0] + kwh, 6), round(slot[1] + hours, 4)
                        agg = kwh_day.setdefault(d, {}).setdefault(key, [0.0, 0.0])
                        agg[0] = round(agg[0] + kwh, 6)
                        agg[1] = round(agg[1] + kwh * (grid or 0), 3)
                else:
                    # No energy counter: keep PSTR as a single reading, never as kWh.
                    add_spot(hist, hid, parse_ts(er.get("last_fetched")), r["now_w"], today)
            if sr:
                ac_sleep, ac_display = as_int(sr.get("ac_sleep")), as_int(sr.get("ac_displaysleep"))
                count = as_int(sr.get("sleep_count"))
                r["sleep_ac"], r["display_ac"] = ac_sleep, ac_display
                r["sleep_off"] = None if ac_sleep is None else int(ac_sleep == 0)
                r["display_long"] = None if ac_display is None else int(ac_display == 0 or ac_display > LONG_DISPLAY_MIN)
                r["never_slept"] = None if count is None or up_days is None else int(count == 0 and up_days >= NEVER_SLEPT_DAYS)
                r["womp"] = None if as_int(sr.get("ac_womp")) is None else int(as_int(sr.get("ac_womp")) == 1)
                blockers = set(prevented_by(sr.get("sleep_now"))) | set(prevented_by(sr.get("displaysleep_now")))
                r["apps"] = sorted(b for b in blockers if b not in MAC_SYSTEM_BLOCKERS)
                r["system"] = sorted(b for b in blockers if b in MAC_SYSTEM_BLOCKERS)

        elif plat == "windows":
            wr = (dex.get("win_power") or {}).get(hid)
            r["kind"] = "laptop" if (det.get("batteries") or (wr and wr.get("power_source"))) else "desktop"
            if wr:
                if (wr.get("power_source") or "") != "AC Power":
                    add_spot(hist, hid, parse_ts(wr.get("last_fetched")), num(wr.get("discharge_watts")), today)
                sleep_s, display_s = as_int(wr.get("sleep_ac_s")), as_int(wr.get("display_ac_s"))
                r["sleep_ac"] = None if sleep_s is None else round(sleep_s / 60)
                r["display_ac"] = None if display_s is None else round(display_s / 60)
                r["sleep_off"] = None if sleep_s is None else int(sleep_s == 0)
                r["display_long"] = None if display_s is None else int(display_s == 0 or display_s > LONG_DISPLAY_MIN * 60)
                sleeps = as_int(wr.get("sleeps_30d"))
                r["never_slept"] = None if sleeps is None else int(sleeps == 0)
                r["fast_startup"] = None if as_int(wr.get("fast_startup")) is None else int(as_int(wr.get("fast_startup")) == 1)

        else:
            lr = (dex.get("lin_power") or {}).get(hid)
            if lr:
                desktop = (lr.get("graphical") or "") == "active"
                r["kind"] = "laptop" if (as_int(lr.get("batteries")) or 0) > 0 else "desktop" if desktop else "server"
                try:
                    names, power = json.loads(lr.get("hwmon_names") or "{}"), json.loads(lr.get("hwmon_power") or "{}")
                except ValueError:
                    names, power = {}, {}
                amd = [num(v) for p, v in power.items() if names.get(p.rsplit("/", 1)[0] + "/name") == "amdgpu"]
                if amd and amd[0]:
                    add_spot(hist, hid, parse_ts(lr.get("last_fetched")), amd[0] / 1e6, today)
                gnome = gnome_value(lr.get("gnome_sleep_ac_type"))
                masked = "masked" in ((lr.get("sleep_target") or ""), (lr.get("suspend_target") or ""))
                if r["kind"] != "server":
                    r["sleep_off"] = int(masked or gnome == "nothing")
                    timeout = as_int(gnome_value(lr.get("gnome_sleep_ac_timeout")))
                    r["sleep_ac"] = 0 if r["sleep_off"] else (round(timeout / 60) if timeout else None)
            else:
                r["kind"] = None

        spot = (hist.get("spot") or {}).get(hid) or {}
        cutoff = (today - dt.timedelta(days=SPOT_DAYS)).isoformat()
        tot = [v for d, v in spot.items() if d >= cutoff]
        n = sum(v[1] for v in tot)
        if n:
            r["spot_w"], r["spot_n"] = round(sum(v[0] for v in tot) / n, 2), n
        mine = host_kwh.get(hid) or {}
        for days in (7, 30):
            since = (today - dt.timedelta(days=days - 1)).isoformat()
            got = [v for d, v in mine.items() if d >= since]
            r[f"kwh{days}"] = round(sum(v[0] for v in got), 4) if got else None
            r[f"hours{days}"] = round(sum(v[1] for v in got), 2) if got else None
        rows.append(r)
        count_into(day, fleet_name, plat, computers=1, sleep_off=r["sleep_off"] or 0,
                   never_slept=r["never_slept"] or 0, measured=int(r["measured"]))

    keep = set(ids)
    for name in ("energy", "host_kwh", "spot", "spot_seen"):
        store = hist.get(name) or {}
        for hid in list(store):
            if hid not in keep:
                del store[hid]
    cutoff = (today - dt.timedelta(days=HOST_KWH_DAYS)).isoformat()
    for hid, days in host_kwh.items():
        host_kwh[hid] = {d: v for d, v in days.items() if d >= cutoff}
    spot_cut = (today - dt.timedelta(days=SPOT_DAYS)).isoformat()
    for hid, days in (hist.get("spot") or {}).items():
        hist["spot"][hid] = {d: v for d, v in days.items() if d >= spot_cut}
    hist["kwh_day"] = trim_daily(kwh_day, TREND_DAYS, today)

    # kWh lives in kwh_day, so merge it into copies and keep the saved trend lean.
    trend = {d: {k: dict(v) for k, v in counts.items()} for d, counts in save_trend(hist, today, day)}
    for d, by_key in hist["kwh_day"].items():
        merged = trend.setdefault(d, {})
        for k, (kwh, co2) in by_key.items():
            merged.setdefault(k, {}).update(kwh=round(kwh, 4), co2=round(co2, 1))
    return {
        "generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"], "history_since": hist["since"],
        "rows": rows, "trend": [[d, trend[d]] for d in sorted(trend)],
        "grid": {"fallback": fallback, "fallback_from": fallback_from, "source": GRID_SOURCE},
        "never_slept_days": NEVER_SLEPT_DAYS, "long_display_min": LONG_DISPLAY_MIN, "spot_days": SPOT_DAYS,
        "virtual": sum(1 for hid, h in hosts.items() if platform_of(h) and is_virtual(h)),
        "reports": {k: bool((info.get(k) or {}).get("found")) for k in ("mac_energy", "mac_sleep", "win_power", "lin_power")},
    }


# ---------------------------------------------------------------- render

BUILDERS = {"lifecycle": build_lifecycle, "software": build_software, "patch": build_patch, "ai": build_ai,
            "connectivity": build_connectivity, "analytics": build_analytics,
            "sustainability": build_sustainability}


# ---------------------------------------------------------------- overview

OVERVIEW_DAYS = 30
# The trend fields each overview card reads, from that page's history.
OVERVIEW_FIELDS = {
    "lifecycle": ("refresh", "battery", "investigate", "old", "scored"),
    "software": ("seats", "idle"),
    "patch": ("app_installs", "app_current", "os_installs", "os_current", "updates"),
    "ai": ("reported", "ai", "act", "high", "med"),
    "connectivity": ("computers", "fresh24", "stale7", "weak", "persistent"),
    "analytics": ("computers", "all_fresh", "disk_known", "disk_on", "agent_current", "failing"),
    "sustainability": ("kwh", "co2", "sleep_off", "never_slept"),
}


def build_overview(snap, page_hists):
    """Reads the daily counts each page just saved, so every number matches that page's trend."""
    today = parse_ts(snap["generated_at"]).date()
    cutoff = (today - dt.timedelta(days=OVERVIEW_DAYS)).isoformat()
    pages = {}
    for page, fields in OVERVIEW_FIELDS.items():
        hist = page_hists.get(page) or {}
        trend = {d: {k: dict(v) for k, v in by_key.items()} for d, by_key in (hist.get("trend") or {}).items()}
        # Sustainability keeps kWh and CO2 apart from its trend.
        for d, by_key in (hist.get("kwh_day") or {}).items():
            for k, (kwh, co2) in by_key.items():
                trend.setdefault(d, {}).setdefault(k, {}).update(kwh=round(kwh, 4), co2=round(co2, 1))
        rows = []
        for d in sorted(trend):
            if d <= cutoff:
                continue
            keep = {k: {f: v[f] for f in fields if v.get(f)} for k, v in trend[d].items()}
            rows.append([d, {k: v for k, v in keep.items() if v}])
        pages[page] = {"since": hist.get("since"), "fields": list(fields), "trend": rows}
    hosts = {}
    for h in snap["hosts"]:
        plat = platform_of(h)
        if plat:
            key = f"{h.get('team_name') or 'Unassigned'}|{plat}"
            hosts[key] = hosts.get(key, 0) + 1
    return {"generated_at": snap["generated_at"], "fleet_url": snap["fleet_url"], "hosts": hosts, "pages": pages}


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
    ap.add_argument("--page", required=True, choices=sorted(BUILDERS) + ["overview"])
    ap.add_argument("--snapshot", required=True)
    ap.add_argument("--templates", required=True)
    ap.add_argument("--history")
    ap.add_argument("--out-html", required=True)
    ap.add_argument("--out-history", help="required for every page but the overview")
    ap.add_argument("--page-history", action="append", default=[], help="overview only: page=path to that page's new history")
    ap.add_argument("--link", action="append", default=[], help="page=URL of that page's artifact")
    args = ap.parse_args()
    snap = load_json(args.snapshot, None)
    if not snap or snap.get("schema") != 1:
        sys.exit("FATAL: snapshot missing or wrong schema")
    links = dict(x.split("=", 1) for x in args.link)
    if args.page == "overview":
        data = build_overview(snap, {p: load_json(path, {}) for p, path in (x.split("=", 1) for x in args.page_history)})
        # Without artifact URLs, link the pages written next to this one.
        data["links"] = {p: links.get(p) or f"{p}.html" for p in OVERVIEW_FIELDS}
    else:
        if not args.out_history:
            sys.exit("FATAL: --out-history is required")
        hist = load_json(args.history, {})
        data = BUILDERS[args.page](snap, hist)
        data["links"] = {"overview": links["overview"]} if links.get("overview") else {}
    with open(args.out_html, "w") as f:
        f.write(render(args.templates, args.page, data))
    if args.page != "overview":
        with open(args.out_history, "w") as f:
            json.dump(hist, f, separators=(",", ":"))
    print(f"{args.page}: html={os.path.getsize(args.out_html)}B"
          + (f" history={os.path.getsize(args.out_history)}B" if args.page != "overview" else ""))


if __name__ == "__main__":
    main()
