#!/usr/bin/env python3
"""Audit Fleet Atlas's specific technical claims against Fleet source code.

Verifies:
  * cron intervals in src/components/background/jobs.ts
  * table names referenced in src/components/data/key-tables.ts
  * endpoint paths referenced in subsystem bodies
  * ErrMissingLicense call sites that back premium claims in src/components/tiers/tiers-data.ts

Resolves the Fleet checkout in order:
  1. $FLEET_REPO_PATH (local-dev override).
  2. Parent-of-parent-of this script, when that directory looks like a Fleet
     repo (has cmd/fleet and server/fleet). This is the in-tree case: Atlas
     lives at <fleet>/tools/fleet-atlas, this script at
     <fleet>/tools/fleet-atlas/scripts/, so ../../.. is the Fleet root.
Falls back to error if neither resolves.

Run locally against an out-of-tree Fleet checkout:
    FLEET_REPO_PATH=~/src/fleet ./scripts/audit-fleet-claims.py

Run in-tree (what CI does) from fleet-atlas root:
    ./scripts/audit-fleet-claims.py

Exits 0 if everything matches, 1 if drift was detected, 2 on config errors.
Prints a readable report either way so you can grep the output.

Colors are off when stdout is not a TTY or when $NO_COLOR is set (CI-friendly).
"""

import os
import re
import sys
import json
from pathlib import Path

ATLAS = Path(__file__).resolve().parent.parent


def _resolve_fleet_path() -> Path:
    env = os.environ.get("FLEET_REPO_PATH")
    if env:
        return Path(env).expanduser()
    # In-tree: tools/fleet-atlas/scripts/audit-fleet-claims.py → fleet root is ../../..
    in_tree_guess = ATLAS.parent.parent
    if (in_tree_guess / "cmd" / "fleet").is_dir() and (
        in_tree_guess / "server" / "fleet"
    ).is_dir():
        return in_tree_guess
    sys.stderr.write(
        "Could not locate the Fleet repo.\n"
        "Set FLEET_REPO_PATH to a local fleetdm/fleet checkout, e.g.:\n"
        "  FLEET_REPO_PATH=~/src/fleet ./scripts/audit-fleet-claims.py\n"
        "(When Atlas is checked in to fleet/tools/fleet-atlas, the script\n"
        "auto-detects the Fleet root as ../../.. of itself.)\n"
    )
    sys.exit(2)


FLEET = _resolve_fleet_path()

_USE_COLOR = sys.stdout.isatty() and not os.environ.get("NO_COLOR")
GREEN = "\033[32m" if _USE_COLOR else ""
RED = "\033[31m" if _USE_COLOR else ""
YELLOW = "\033[33m" if _USE_COLOR else ""
DIM = "\033[2m" if _USE_COLOR else ""
RESET = "\033[0m" if _USE_COLOR else ""


def section(title):
    print(f"\n{YELLOW}━━ {title} ━━{RESET}")


def ok(msg):
    print(f"  {GREEN}✓{RESET} {msg}")


def mismatch(msg):
    print(f"  {RED}✗{RESET} {msg}")


def info(msg):
    print(f"  {DIM}•{RESET} {msg}")


# ─────────────────────────────────────────────────────────────────────────────
# 1. CRON INTERVALS
# ─────────────────────────────────────────────────────────────────────────────


def extract_fleet_crons():
    """Pull every (CronX, defaultInterval) pair from cmd/fleet/cron.go."""
    src = (FLEET / "cmd/fleet/cron.go").read_text()
    pat = re.compile(
        r"name\s*=\s*string\(fleet\.(Cron\w+)\)(.{0,500}?)defaultInterval\s*=\s*([0-9]+)\s*\*\s*time\.(\w+)",
        re.DOTALL,
    )
    out = {}
    for m in pat.finditer(src):
        name = m.group(1)
        n = int(m.group(3))
        unit = m.group(4)
        minutes = {"Second": n / 60, "Minute": n, "Hour": n * 60}.get(unit)
        if minutes is None:
            continue
        out.setdefault(name, minutes)  # keep first occurrence
    return out


def constant_to_cron_name():
    """Map CronX constant name -> runtime cron name string."""
    src = (FLEET / "server/fleet/cron_schedules.go").read_text()
    pat = re.compile(r"(Cron\w+)\s+CronScheduleName\s*=\s*\"([^\"]+)\"")
    return {m.group(1): m.group(2) for m in pat.finditer(src)}


def extract_atlas_cron_intervals():
    """Pull (name, intervalMinutes) pairs from jobs.ts."""
    src = (ATLAS / "src/components/background/jobs.ts").read_text()
    pat = re.compile(
        r'name:\s*"([^"]+)",\s*intervalMinutes:\s*([0-9]+(?:\.[0-9]+)?(?:\s*\*\s*[0-9]+)?)',
    )
    out = {}
    for m in pat.finditer(src):
        name = m.group(1)
        raw = m.group(2).replace(" ", "")
        if "*" in raw:
            a, b = raw.split("*")
            minutes = float(a) * float(b)
        else:
            minutes = float(raw)
        out[name] = minutes
    return out


# Crons whose hard default is overridden at admin-configuration time.
# Atlas intentionally lists the common configured value, not the default.
INTENTIONAL_INTERVAL_OVERRIDES = {
    "automations",  # default 24h, overridden by WebhookSettings.Interval
}


def audit_crons():
    section("Cron intervals")
    fleet_by_const = extract_fleet_crons()
    const_to_name = constant_to_cron_name()
    fleet_by_name = {const_to_name[c]: m for c, m in fleet_by_const.items() if c in const_to_name}
    atlas = extract_atlas_cron_intervals()

    mismatches = 0
    for name, atlas_min in sorted(atlas.items()):
        base = name.split(" (")[0]  # strip "apple_mdm_apns_pusher (legacy)" annotation
        if base not in fleet_by_name:
            info(f"{name}: no defaultInterval in cron.go (may be admin-configurable). skipping")
            continue
        truth = fleet_by_name[base]
        if abs(truth - atlas_min) < 0.01:
            ok(f"{name}: {_pretty(atlas_min)}")
        elif base in INTENTIONAL_INTERVAL_OVERRIDES:
            info(
                f"{name}: atlas says {_pretty(atlas_min)}, Fleet default {_pretty(truth)} — intentional (admin-configured)"
            )
        else:
            mismatches += 1
            mismatch(
                f"{name}: atlas says {_pretty(atlas_min)}, Fleet says {_pretty(truth)}"
            )

    for name in sorted(fleet_by_name):
        if name in atlas or f"{name} (legacy)" in atlas:
            continue
        mismatch(f"MISSING FROM ATLAS: {name} (Fleet default {_pretty(fleet_by_name[name])})")
        mismatches += 1

    return mismatches


def _pretty(mins):
    if mins < 1:
        return f"{int(mins * 60)}s"
    if mins < 60:
        return f"{mins:g}m"
    if mins < 60 * 24:
        return f"{mins / 60:g}h"
    return f"{mins / (60 * 24):g}d"


# ─────────────────────────────────────────────────────────────────────────────
# 2. TABLE NAMES
# ─────────────────────────────────────────────────────────────────────────────


def extract_atlas_tables():
    src = (ATLAS / "src/components/data/key-tables.ts").read_text()
    return re.findall(r'name:\s*"([a-z_][a-z0-9_]+)"', src)


def extract_fleet_table_names():
    """Union of CREATE TABLE statements across all migrations + schema.sql."""
    tables = set()
    schema = FLEET / "server/datastore/mysql/schema.sql"
    if schema.exists():
        for m in re.finditer(r"CREATE TABLE `?([a-z_][a-z0-9_]+)`?", schema.read_text()):
            tables.add(m.group(1))
    mig_dir = FLEET / "server/datastore/mysql/migrations/tables"
    for p in mig_dir.glob("*.go"):
        if p.name.endswith("_test.go"):
            continue
        for m in re.finditer(r"CREATE TABLE (?:IF NOT EXISTS )?`?([a-z_][a-z0-9_]+)`?", p.read_text()):
            tables.add(m.group(1))
    return tables


def audit_tables():
    section("Key tables")
    atlas = extract_atlas_tables()
    fleet = extract_fleet_table_names()
    mismatches = 0
    for name in atlas:
        if name in fleet:
            ok(f"{name}")
        else:
            mismatches += 1
            mismatch(f"{name}: NOT FOUND in Fleet migrations or schema.sql")
    return mismatches


# ─────────────────────────────────────────────────────────────────────────────
# 3. ENDPOINT PATHS
# ─────────────────────────────────────────────────────────────────────────────


def extract_atlas_endpoints():
    """Pull `/api/...` paths from subsystem bodies + tracers."""
    paths = set()
    for rel in ["src/components/subsystems/subsystems.ts", "src/components/tracer/tracers.ts"]:
        src = (ATLAS / rel).read_text()
        for m in re.finditer(r"(?:POST |GET |/)?(/api/[a-zA-Z0-9/_{}.-]+)", src):
            p = m.group(1).rstrip(" .,);\"'`\n")
            # Skip bare prefixes (ending in / mid-mention of /api/osquery/ etc.)
            if p.endswith("/") or p.count("/") < 3:
                continue
            paths.add(p)
    return paths


def extract_fleet_registered_endpoints():
    """Union of API path literals across every known handler.go in the repo."""
    paths = set()
    candidates = [
        FLEET / "server/service/handler.go",
        FLEET / "server/mdm/android/service/handler.go",
    ]
    # pick up any other handler.go we might not know about
    for p in FLEET.rglob("handler.go"):
        candidates.append(p)
    seen = set()
    for p in candidates:
        if p in seen or not p.exists():
            continue
        seen.add(p)
        for m in re.finditer(r'"(/api/[^"]+)"', p.read_text()):
            paths.add(m.group(1))
    return paths


def _normalize_version(p):
    """Fleet's handler.go uses _version_ as the version placeholder, so
    normalize /api/v1/… and /api/latest/… against that for comparison."""
    return re.sub(r"/api/(v\d+|latest|_version_)/", "/api/_version_/", p)


def _path_to_regex(p):
    # convert {foo} → [^/]+ for flexible matching
    return "^" + re.sub(r"\{[^}]+\}", r"[^/]+", p) + "$"


def audit_endpoints():
    section("Endpoint paths")
    atlas = extract_atlas_endpoints()
    fleet = {_normalize_version(f) for f in extract_fleet_registered_endpoints()}
    mismatches = 0
    for p in sorted(atlas):
        norm = _normalize_version(p)
        if norm in fleet:
            ok(p)
            continue
        if any(re.match(_path_to_regex(norm), f) for f in fleet):
            ok(p + "  (path-param match)")
            continue
        mismatches += 1
        mismatch(f"{p}: not registered in handler.go (checked as {norm})")
    return mismatches


# ─────────────────────────────────────────────────────────────────────────────
# 4. ErrMissingLicense PREMIUM CLAIMS
# ─────────────────────────────────────────────────────────────────────────────


def premium_citations_in_atlas():
    """Each tiers-data note referencing a specific Fleet file + anchor word."""
    src = (ATLAS / "src/components/tiers/tiers-data.ts").read_text()
    anchors = re.findall(
        r"note:\s*\"[^\"]*?\b(InstallSoftwareTitle|SelfServiceInstallSoftwareTitle|UploadSoftwareInstaller|ListFleetMaintainedApps|validateActivation|PSSO)\b",
        src,
    )
    return anchors


def audit_premium_claims():
    section("Premium-only claim anchors")
    anchors = premium_citations_in_atlas()
    files_to_check = list((FLEET / "server/service").glob("*.go")) + list(
        (FLEET / "server/mdm").rglob("*.go")
    )
    all_src = "\n".join(
        p.read_text() for p in files_to_check if not p.name.endswith("_test.go")
    )
    mismatches = 0
    for a in sorted(set(anchors)):
        # Look at every occurrence of the anchor and see if ErrMissingLicense
        # or NewLicenseErrorWithCause appears within 2000 chars after it (the
        # typical body of a wrapper method).
        occurrences = [i for i in range(len(all_src)) if all_src.startswith(a, i)]
        gated = any(
            "ErrMissingLicense" in all_src[i : i + 2000]
            or "NewLicenseErrorWithCause" in all_src[i : i + 2000]
            for i in occurrences
        )
        if gated:
            ok(f"{a}: still gated by license check")
        elif not occurrences:
            mismatches += 1
            mismatch(f"{a}: symbol not found in Fleet source at all")
        else:
            mismatches += 1
            mismatch(
                f"{a}: found in Fleet but no license gate within 2000 chars (verify by hand)"
            )
    return mismatches


# ─────────────────────────────────────────────────────────────────────────────
# MAIN
# ─────────────────────────────────────────────────────────────────────────────


def main():
    if not FLEET.exists():
        print(f"{RED}Fleet repo not found at {FLEET}. Set FLEET_REPO_PATH.{RESET}")
        sys.exit(2)
    print(f"{DIM}Fleet repo:  {FLEET}{RESET}")
    print(f"{DIM}Atlas repo:  {ATLAS}{RESET}")

    total = 0
    total += audit_crons()
    total += audit_tables()
    total += audit_endpoints()
    total += audit_premium_claims()

    print()
    if total == 0:
        print(f"{GREEN}All claims verified against Fleet source.{RESET}")
    else:
        print(f"{RED}{total} drift(s) detected. Fix in Atlas or verify against Fleet.{RESET}")
    sys.exit(0 if total == 0 else 1)


if __name__ == "__main__":
    main()
