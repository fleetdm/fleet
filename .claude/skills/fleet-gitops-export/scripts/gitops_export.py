#!/usr/bin/env python3
"""Export a live Fleet instance into a GitOps repo laid out by `fleetctl new`.

Usage:
  gitops_export.py preflight --dir DIR [--org-name NAME] [--force]
  gitops_export.py scaffold  --dir DIR --org-name NAME [--force]
  gitops_export.py export    --dir DIR
  gitops_export.py place     --dir DIR
  gitops_export.py validate  --dir DIR [--env-file FILE] [--offline] [--no-dry-run]
  gitops_export.py report    --dir DIR
  gitops_export.py all       --dir DIR --org-name NAME [--force] [--env-file FILE]
  gitops_export.py clean     --dir DIR

Connection flags (any command): --fleetctl PATH, --config FILE, --context NAME.
They're remembered for the directory after the first command.

Read-only against Fleet: every call is a GET through `fleetctl api`, and the
only `fleetctl gitops` run is a --dry-run. Work files (the plan, downloaded
artifacts, validation results) go in an owner-only temp folder outside the
repo, printed by each command (--work picks another one; pass it to every
command). None of them hold secret values, but they do hold copies of
profiles and scripts; `clean` deletes the folder.

Values Fleet can't return (FLEET_SECRET_* variables, masked tokens) come from
the repo's .env, or --env-file, and then the environment. Nothing prints them.

Needs Python 3.9+ and fleetctl 4.84.0+. PyYAML or Ruby enables the YAML checks.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from checks import Checker  # noqa: E402
from collect import Collector  # noqa: E402
from fleet_api import API, Fleetctl, FleetctlError  # noqa: E402
from layout import Layout  # noqa: E402
from placer import Placer  # noqa: E402
from planner import Planner  # noqa: E402

MIN_FLEETCTL = (4, 84, 0)
REPORT_STATIC_LIMIT = 25  # validator findings listed in the report; the rest stay in validation.json
SINGLE = {"org settings", "agent options", "ADE profile", "setup script"}


def work_dir(target: Path, override: str | None) -> Path:
    if override:
        w = Path(override)
    else:
        digest = hashlib.sha1(str(target.resolve()).encode()).hexdigest()[:10]
        w = Path(tempfile.gettempdir()) / f"fleet-gitops-export-{target.resolve().name}-{digest}"
    # It holds copies of profiles and scripts as Fleet stores them, and its default name is
    # predictable in a shared temp folder: keep it owner-only and refuse one someone else made.
    if w.is_symlink():
        raise SystemExit(f"FAIL {w} is a symlink; pass --work with a folder you own")
    w.mkdir(mode=0o700, parents=True, exist_ok=True)
    if hasattr(os, "getuid") and w.stat().st_uid != os.getuid():
        raise SystemExit(f"FAIL {w} belongs to another user; pass --work with a folder you own")
    os.chmod(w, 0o700)
    return w


def session(args) -> tuple[Fleetctl, Path, Path]:
    target = Path(args.dir).expanduser()
    work = work_dir(target, args.work)
    saved_path = work / "session.json"
    saved = json.loads(saved_path.read_text()) if saved_path.exists() else {}
    conn = {k: getattr(args, k) or saved.get(k) for k in ("fleetctl", "config", "context")}
    saved_path.write_text(json.dumps(conn, indent=2))
    return Fleetctl(conn["fleetctl"] or "fleetctl", conn["config"], conn["context"]), target, work


def need(work: Path, name: str, step: str) -> Path:
    p = work / name
    if not p.exists():
        raise SystemExit(f"FAIL {p} doesn't exist; run `{step}` first, or pass --work with the folder an earlier command printed")
    return p


def dry_run_env_file(args, target: Path) -> Path | None:
    if args.env_file:
        p = Path(args.env_file).expanduser()
        if not p.is_file():
            raise SystemExit(f"FAIL --env-file {p} doesn't exist")
        return p
    p = target / ".env"
    return p if p.is_file() else None


def parse_version(v: str):
    m = re.match(r"v?(\d+)\.(\d+)\.(\d+)", v or "")
    return tuple(int(x) for x in m.groups()) if m else None


def say(msg=""):
    print(msg, flush=True)


# ------------------------------------------------------------ commands

def cmd_preflight(args) -> int:
    fc, target, work = session(args)
    problems, info = [], {}
    try:
        info["fleetctl_version"] = fc.version()
    except (OSError, FleetctlError) as e:
        say(f"FAIL fleetctl: {e}. Install it: https://fleetdm.com/guides/fleetctl#installing-fleetctl")
        return 1
    v = parse_version(info["fleetctl_version"])
    if not fc.has_new() or (v and v < MIN_FLEETCTL):
        problems.append(f"fleetctl {info['fleetctl_version']} has no `fleetctl new`; upgrade to 4.84.0 or later")
    try:
        info["address"] = fc.address()
        api = API(fc)
        me = api.json("/api/latest/fleet/me").get("user", {})
        info["user"] = me.get("email")
        info["global_role"] = me.get("global_role")
        info["server_version"] = api.json("/api/latest/fleet/version").get("version")
        cfg = api.json("/api/latest/fleet/config")
    except FleetctlError as e:
        say(f"FAIL can't reach Fleet with fleetctl's credentials: {e}")
        say("     Log in with `fleetctl login` (or `fleetctl config set --address URL --token TOKEN`), then retry.")
        return 1
    if info["global_role"] != "admin":
        problems.append(f"{info['user']} isn't a global admin (role: {info['global_role']}); the export needs one to read secrets-adjacent settings and every fleet")
    sv = parse_version(info["server_version"] or "")
    if v and sv and v[:2] != sv[:2]:
        info["version_warning"] = f"fleetctl {info['fleetctl_version']} and the server ({info['server_version']}) differ; CI installs fleetctl to match the server, so a mismatch can hide or invent dry-run differences"
    lic = (cfg.get("license") or {}).get("tier")
    mdm = cfg.get("mdm") or {}
    info.update({
        "server_url": (cfg.get("server_settings") or {}).get("server_url"),
        "license": lic,
        "org_name": (cfg.get("org_info") or {}).get("org_name"),
        "apple_mdm": mdm.get("enabled_and_configured"),
        "windows_mdm": mdm.get("windows_enabled_and_configured"),
        "android_mdm": mdm.get("android_enabled_and_configured"),
        "gitops": cfg.get("gitops"),
        "python": sys.version.split()[0],
        "work_dir": str(work),
    })
    if target.exists() and any(target.iterdir()) and not args.force:
        problems.append(f"{target} isn't empty; pick an empty directory, or pass --force once the user has agreed to write into it")
    say(json.dumps(info, indent=2, ensure_ascii=False))
    for p in problems:
        say(f"FAIL {p}")
    if info.get("version_warning"):
        say(f"WARN {info['version_warning']}")
    return 1 if problems else 0


def cmd_scaffold(args) -> int:
    fc, target, work = session(args)
    if not args.org_name:
        say("FAIL --org-name is required")
        return 1
    if target.exists() and any(target.iterdir()) and not args.force:
        say(f"FAIL {target} isn't empty; pass --force only if the user agreed to write into it")
        return 1
    force = target.exists()
    out = fc.new(str(target), args.org_name, force=force)
    pristine = work / "pristine"
    if pristine.exists():
        shutil.rmtree(pristine)
    shutil.copytree(target, pristine, ignore=shutil.ignore_patterns(".git"))
    (work / "placed.json").unlink(missing_ok=True)
    layout = Layout.scan(str(target))
    layout.save(work / "layout.json")
    say(out.split("\nNext steps")[0].strip())
    say(f"layout: {len(layout.fleet_templates)} fleet templates, {len(layout.globs)} globs, platform folders under {layout.platform_root}/")
    return 0


def cmd_export(args) -> int:
    fc, target, work = session(args)
    layout = Layout.load(need(work, "layout.json", "scaffold"))
    snap = Collector(API(fc), log=lambda m: say(f"  {m}")).collect()
    blobs = work / "blobs"
    if blobs.exists():
        shutil.rmtree(blobs)
    plan = Planner(snap, layout, work).build()
    (work / "plan.json").write_text(json.dumps(plan, indent=2, ensure_ascii=False), encoding="utf-8")
    say(f"plan: {len(plan['files'])} files, {len(plan['fleets'])} fleet files, {len(plan['env'])} environment variables")
    reviews = [n for n in plan["notes"] if n["level"] == "review"]
    gaps = [n for n in plan["notes"] if n["level"] == "gap"]
    for n in reviews:
        say(f"REVIEW {n['scope']}: {n['item']}: {n['message']}")
    for n in gaps:
        say(f"GAP {n['scope']}: {n['item']}: {n['message']}")
    say(f"plan written to {work / 'plan.json'}")
    return 0


def cmd_place(args) -> int:
    fc, target, work = session(args)
    layout = Layout.load(need(work, "layout.json", "scaffold"))
    plan = json.loads(need(work, "plan.json", "export").read_text(encoding="utf-8"))
    try:
        Placer(plan, layout, work).place()
    except ValueError as e:
        say(f"FAIL {e}")
        return 1
    new_dirs = sorted({os.path.dirname(f["path"]) for f in plan["files"].values() if f.get("origin") == "new"})
    say(f"placed {len(plan['files'])} files into {target}")
    for d in new_dirs:
        say(f"NEW FOLDER {d}/ (the fleetctl new layout has no folder for this kind of file)")
    return 0


def cmd_validate(args) -> int:
    fc, target, work = session(args)
    plan = json.loads(need(work, "plan.json", "export").read_text(encoding="utf-8"))
    env_file = None if args.no_dry_run else dry_run_env_file(args, target)
    ch = Checker(target, plan)
    ch.run_validator(offline=args.offline)
    if ch.load_repo():
        ch.references()
        ch.reconcile()
    if not args.no_dry_run:
        say(f"dry-run variables from {env_file} and the environment" if env_file else "dry-run variables from the environment (no .env in the repo)")
        ch.dry_run(API(fc), fc, env_file)
    res = ch.summary()
    (work / "validation.json").write_text(json.dumps(res, indent=2, ensure_ascii=False), encoding="utf-8")
    for section in ("static", "references", "reconcile"):
        for r in res[section]:
            say(f"{r['level']} [{section}] {r['message']}")
    dr = res.get("dry_run") or {}
    if dr:
        say(f"dry run: exit {dr['exit_code']}, {len(dr['changes'])} change lines")
        for n in dr["notes"]:
            say(f"NOTE {n}")
    say("PASS" if res["passed"] else "FAIL")
    return 0 if res["passed"] else 1


def cmd_report(args) -> int:
    fc, target, work = session(args)
    plan = json.loads(need(work, "plan.json", "export").read_text(encoding="utf-8"))
    val_path = work / "validation.json"
    val = json.loads(val_path.read_text(encoding="utf-8")) if val_path.exists() else None
    text = render_report(target, plan, val)
    (work / "report.md").write_text(text, encoding="utf-8")
    say(text)
    return 0


def cmd_clean(args) -> int:
    fc, target, work = session(args)
    shutil.rmtree(work)
    say(f"removed {work}")
    return 0


def cmd_all(args) -> int:
    for step in (cmd_preflight, cmd_scaffold, cmd_export, cmd_place):
        rc = step(args)
        if rc:
            return rc
    rc = cmd_validate(args)
    cmd_report(args)
    return rc


# ------------------------------------------------------------ report

def render_report(target: Path, plan: dict, val: dict | None) -> str:
    srv = plan["server"]
    lines = [f"# Fleet GitOps export: {target}", "",
             f"Fleet {srv['version']} ({'Premium' if srv['premium'] else 'Free'}), exported as {srv['user'].get('email')}.", ""]
    lines += ["## Exported", "", "| File | Scope | Contents |", "|---|---|---|"]
    scope_file = {}
    for f in plan["fleets"]:
        scope = {"global": "Global" if srv["premium"] else "Global", "unassigned": "Unassigned"}.get(f["key"], f.get("name") or f["sections"].get("name"))
        scope_file[scope] = f["file"]
    if srv["premium"]:
        scope_file["Unassigned controls"] = "default.yml"
    for scope, counts in plan["counts"].items():
        file = scope_file.get(scope, "default.yml" if scope in ("Global", "Unassigned") else "?")
        if scope == "Unassigned":
            file = "fleets/unassigned.yml + default.yml (controls)"
        summary = ", ".join(k if k in SINGLE else f"{n} {k}" for k, n in sorted(counts.items(), key=lambda kv: kv[0].lower()))
        lines.append(f"| `{file}` | {scope} | {summary} |")
    review = [n for n in plan["notes"] if n["level"] == "review"]
    gaps = [n for n in plan["notes"] if n["level"] in ("gap", "warn")]
    info = [n for n in plan["notes"] if n["level"] == "info"]
    new_dirs = sorted({os.path.dirname(f["path"]) for f in plan["files"].values() if f.get("origin") == "new"})
    if new_dirs:
        lines += ["", "## Folders added to the layout", "",
                  "`fleetctl new` has no folder for these kinds of files, so they follow its `platforms/<os>/<kind>/` pattern:", ""]
        lines += [f"- `{d}/`" for d in new_dirs]
    if review:
        lines += ["", "## Placement choices to check", ""] + [f"- **{n['scope']}**, `{n['item']}`: {n['message']}" for n in review]
    lines += ["", "## Not represented in GitOps", ""]
    lines += [f"- **{n['scope']}**, {n['item']}: {n['message']}" for n in gaps] or ["- Nothing."]
    if info:
        lines += ["", "## Notes", ""] + [f"- **{n['scope']}**, {n['item']}: {n['message']}" for n in info]
    lines += ["", "## Environment variables", "",
              "Set each as a repository secret (GitHub) or CI/CD variable (GitLab). `.env.example` and the README list them too.", "",
              "| Variable | Used in | What it is |", "|---|---|---|"]
    for e in plan["env"]:
        lines.append(f"| `{e['name']}` | {', '.join(f'`{u}`' for u in e['used_in'])} | {e['description']} |")
    if val:
        caveat = ("`fleetctl gitops --dry-run` lists most sections as \"would've applied\" without comparing them, and only reports "
                  "deletions for policies, reports, labels, software packages, custom host vitals, Android certificates, and fleets. "
                  "The repo-vs-Fleet check covers the rest by resolving every path and glob and comparing the result with Fleet, item by item and field by field.")
        lines += ["", "## Validation", "", caveat, ""]
        for section, title in (("references", "YAML and references"), ("reconcile", "Repo vs. Fleet")):
            fails = [r for r in val[section] if r["level"] == "FAIL"]
            lines.append(f"- {title}: {'pass' if not fails else f'{len(fails)} failure(s)'}")
            lines += [f"  - {r['message']}" for r in val[section]]
        static = val.get("static") or []
        lines.append(f"- fleet-gitops validator: {len([r for r in static if r['level'] == 'FAIL'])} FAIL, {len([r for r in static if r['level'] == 'WARN'])} WARN")
        lines += [f"  - {r['level']}: {r['message']}" for r in static[:REPORT_STATIC_LIMIT]]
        if len(static) > REPORT_STATIC_LIMIT:
            lines.append(f"  - …and {len(static) - REPORT_STATIC_LIMIT} more in validation.json")
        dr = val.get("dry_run") or {}
        if dr:
            lines.append(f"- Dry run: exit code {dr['exit_code']}, {len(dr['changes'])} change line(s)")
            lines += [f"  - {c}" for c in dr["changes"]]
            lines += [f"  - note: {n}" for n in dr["notes"]]
        lines.append(f"- Overall: **{'PASS' if val['passed'] else 'FAIL'}**")
    files = ["default.yml"] + sorted(f["file"] for f in plan["fleets"] if f["file"] != "default.yml")
    lines += ["", "## Rerun the dry run", "",
              "Export the variables above with their real values (for example from a local `.env` that's never committed), then:", "",
              "```bash", f"cd {target}", "set -a; . ./.env; set +a",
              "fleetctl gitops " + " ".join(f"-f {f}" for f in files) + " --delete-other-fleets --dry-run", "```", ""]
    return "\n".join(lines)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("preflight", "scaffold", "export", "place", "validate", "report", "all", "clean"):
        p = sub.add_parser(name)
        p.add_argument("--dir", required=True, help="target GitOps repo directory")
        p.add_argument("--work", help="work folder (default: a temp folder derived from --dir)")
        p.add_argument("--fleetctl", help="fleetctl binary (default: fleetctl on PATH)")
        p.add_argument("--config", help="fleetctl config file")
        p.add_argument("--context", help="fleetctl config context")
        if name in ("preflight", "scaffold", "all"):
            p.add_argument("--org-name")
            p.add_argument("--force", action="store_true", help="write into a non-empty directory (only with the user's OK)")
        if name in ("validate", "all"):
            p.add_argument("--env-file", help="values Fleet can't return, for the dry run (default: the repo's .env)")
            p.add_argument("--offline", action="store_true", help="skip downloads in the fleet-gitops validator")
            p.add_argument("--no-dry-run", action="store_true")
    args = ap.parse_args()
    for k in ("org_name", "force", "env_file", "offline", "no_dry_run"):
        if not hasattr(args, k):
            setattr(args, k, None)
    rc = {"preflight": cmd_preflight, "scaffold": cmd_scaffold, "export": cmd_export, "place": cmd_place,
          "validate": cmd_validate, "report": cmd_report, "all": cmd_all, "clean": cmd_clean}[args.cmd](args)
    sys.exit(rc)


if __name__ == "__main__":
    main()
