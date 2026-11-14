#!/usr/bin/env python3
"""The latest releases from outside Fleet, so the pages measure computers against
what vendors shipped rather than the newest version the fleet happens to run.

  macOS    SOFA, sofafeed.macadmins.io: Apple's releases with their dates, which
           major versions Apple still updates, and the newest Apple silicon
           generation among Mac models
  Windows  Microsoft's Security Update Guide, api.msrc.microsoft.com: the newest
           security build of each Windows release
  fleetd   Fleet's update server, updates.fleetdm.com: the stable channel

Only issues GET requests. Each source fails on its own, and a page whose source
is missing falls back to the newest version the fleet runs. run.py calls fetch();
run this file on its own to see what the sources return:

  vendor.py --out vendor.json
"""
import argparse
import datetime as dt
import json
import re
import urllib.request

SOFA_URL = "https://sofafeed.macadmins.io/v2/macos_data_feed.json"
MSRC_UPDATES_URL = "https://api.msrc.microsoft.com/cvrf/v3.0/updates"
TUF_TARGETS_URL = "https://updates.fleetdm.com/targets.json"
TIMEOUT = 120
# "Windows 11 Version 25H2 for x64-based Systems" is the release Fleet reports as "Windows 11 25H2".
WINDOWS_PRODUCT = re.compile(r"(Windows 1[01]) Version (\d\dH\d) for x64-based Systems")
FLEETD_TARGETS = {"orbit": "orbit", "osqueryd": "osquery", "desktop": "desktop"}


def get_json(url):
    req = urllib.request.Request(url, headers={"Accept": "application/json", "User-Agent": "fleet-dex-dashboards"})
    with urllib.request.urlopen(req, timeout=TIMEOUT) as r:
        return json.load(r)


def numeric(v):
    return [int(x) for x in re.findall(r"\d+", v or "")]


def macos():
    d = get_json(SOFA_URL)
    majors, releases, first = {}, {}, {}
    for o in d.get("OSVersions") or []:
        latest = o.get("Latest") or {}
        if not (ver := latest.get("ProductVersion")):
            continue
        major = ver.split(".")[0]
        majors[major] = {"name": o.get("OSVersion") or "", "latest": ver, "build": latest.get("Build") or "",
                         "released": (latest.get("ReleaseDate") or "")[:10]}
        dates = [r["ReleaseDate"][:10] for r in o.get("SecurityReleases") or [] if r.get("ReleaseDate")]
        first[major] = min(dates) if dates else majors[major]["released"]
        for r in o.get("SecurityReleases") or []:
            if r.get("ProductVersion") and r.get("ReleaseDate"):
                releases.setdefault(r["ProductVersion"], r["ReleaseDate"][:10])
    if not majors:
        raise ValueError("no macOS releases in the feed")
    # SOFA has no support flag. Apple stops updating its oldest major version when a
    # new one ships, so a major is still updated if it's had a release since then.
    newest_first = first[max(majors, key=int)]
    for m in majors.values():
        m["supported"] = m["released"] >= newest_first
    gens = [int(m.group(1)) for v in (d.get("Models") or {}).values()
            if (m := re.search(r"\bM(\d+)\b", v.get("MarketingName") or ""))]
    return {"majors": majors, "releases": releases, "chip_gen": max(gens) if gens else None}


def windows():
    # Monthly documents; the newest one has every Windows release's latest security build.
    docs = get_json(MSRC_UPDATES_URL).get("value") or []
    doc = max(docs, key=lambda u: u.get("InitialReleaseDate") or "")
    d = get_json(doc["CvrfUrl"])
    names = {}

    def walk(node):
        for p in node.get("FullProductName") or []:
            names[p["ProductID"]] = p["Value"]
        for b in node.get("Branch") or []:
            walk(b)
    walk(d.get("ProductTree") or {})
    builds = {}
    for vuln in d.get("Vulnerability") or []:
        for r in vuln.get("Remediations") or []:
            if not (fixed := r.get("FixedBuild")):
                continue
            for pid in r.get("ProductID") or []:
                if m := WINDOWS_PRODUCT.fullmatch(names.get(pid, "")):
                    title = f"{m.group(1)} {m.group(2)}"
                    if title not in builds or numeric(fixed) > numeric(builds[title]):
                        builds[title] = fixed
    if not builds:
        raise ValueError(f"no Windows builds in {doc.get('ID')}")
    return {"month": doc["ID"], "released": doc["InitialReleaseDate"][:10], "releases": builds}


def fleetd():
    targets = get_json(TUF_TARGETS_URL)["signed"]["targets"]
    out = {}
    for path, meta in targets.items():
        parts = path.split("/")
        name = FLEETD_TARGETS.get(parts[0])
        v = (meta.get("custom") or {}).get("version")
        if name and len(parts) > 2 and parts[2] == "stable" and v and (name not in out or numeric(v) > numeric(out[name])):
            out[name] = v
    if not out:
        raise ValueError("no stable versions on the update server")
    return out


def fetch():
    out = {"fetched_at": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")}
    for name, source in (("macos", macos), ("windows", windows), ("fleetd", fleetd)):
        try:
            out[name] = source()
        except Exception as e:
            out[name] = {"error": f"{type(e).__name__}: {e}"[:300]}
    return out


def summary(v):
    def one(name, ok):
        s = v.get(name) or {}
        return f"{name}={'unavailable' if 'error' in s else ok(s)}"
    return "vendor: " + " ".join([
        one("macos", lambda s: max((m["latest"] for m in s["majors"].values()), key=numeric)),
        one("windows", lambda s: s["month"]),
        one("fleetd", lambda s: s.get("orbit", "?")),
    ])


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    args = ap.parse_args()
    v = fetch()
    with open(args.out, "w") as f:
        json.dump(v, f, indent=1)
    print(summary(v))
    for name in ("macos", "windows", "fleetd"):
        if "error" in v[name]:
            print(f"{name}: {v[name]['error']}")


if __name__ == "__main__":
    main()
