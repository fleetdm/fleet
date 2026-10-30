"""Validate an exported repo: YAML parses, references resolve, nothing is orphaned,
the repo defines exactly what Fleet has, and a fleetctl dry run is clean.

`fleetctl gitops --dry-run` prints "would've applied N" for most sections
without diffing them, and only reports deletions for policies, reports,
labels, software packages, custom host vitals, Android certificates, and
fleets. The reconcile check fills that gap: it resolves every path and glob
on disk and compares the result with what Fleet has, item by item and field
by field (content hashes for scripts, profiles, and DDM assets).
"""

from __future__ import annotations

import glob as globlib
import hashlib
import importlib.util
import json
import os
import plistlib
import re
import subprocess
import sys
from pathlib import Path

from fleet_api import API, FleetctlError

F = "/api/latest/fleet"
PLACEHOLDER = "fleet-gitops-export-dry-run-placeholder"
MASK = "********"

PATH_KEYS = {"path", "paths", "package_path", "activation", "apple_setup_assistant", "macos_script",
             "end_user_license_agreement", "windows_eula", "org_logo_path_light_mode", "org_logo_path_dark_mode"}
IGNORED_ROOTS = (".github/", ".agents/", ".claude/", ".git/")
IGNORED_FILES = {"README.md", "AGENTS.md", "CLAUDE.md", "GEMINI.md", ".gitignore", ".gitlab-ci.yml", ".env.example", ".env"}


# ------------------------------------------------------------ YAML loading

def yaml_backend():
    if importlib.util.find_spec("yaml"):
        return "pyyaml"
    try:
        subprocess.run(["ruby", "-ryaml", "-e", "1"], check=True, capture_output=True, timeout=20)
        return "ruby"
    except (OSError, subprocess.SubprocessError):
        return None


def load_yaml(text: str, backend: str):
    if backend == "pyyaml":
        import yaml
        return yaml.safe_load(text)
    script = "require 'yaml'; require 'json'; puts JSON.generate(YAML.safe_load(STDIN.read, aliases: false))"
    r = subprocess.run(["ruby", "-e", script], input=text, capture_output=True, text=True, timeout=60)
    if r.returncode:
        raise ValueError((r.stderr.strip().splitlines() or ["ruby YAML parse failed"])[-1])
    return json.loads(r.stdout)


KIND_NAMES = {"policies": "policy", "reports": "report", "labels": "label", "scripts": "script",
              "profiles": "profile", "assets": "DDM asset", "software": "software"}


def _sha(b: bytes) -> str:
    return hashlib.sha256(b).hexdigest()


def _norm(v):
    """Comparable form: empty values (None, "", false, 0, [], {}) are equal, lists are unordered."""
    if isinstance(v, list):
        items = [x for x in (_norm(i) for i in v) if x is not None]
        return sorted(items, key=lambda x: json.dumps(x, sort_keys=True)) or None
    if isinstance(v, dict):
        d = {k: x for k, x in ((k, _norm(i)) for k, i in v.items()) if x is not None}
        return d or None
    if v is None or v is False or v == "" or (isinstance(v, (int, float)) and not isinstance(v, bool) and v == 0):
        return None
    return v if isinstance(v, bool) else str(v)


def _short(v) -> str:
    text = json.dumps(v, ensure_ascii=False, default=str)
    return text if len(text) <= 80 else text[:77] + "..."


def _package_settings(item: dict, entry: dict) -> dict:
    """A package's effective settings: its own, else the fleet-level entry's (as fleetctl merges them)."""
    f = {"self_service": item.get("self_service") or entry.get("self_service"),
         "categories": item.get("categories") or entry.get("categories")}
    labels = {k: item[k] for k in ("labels_include_any", "labels_exclude_any", "labels_include_all") if item.get(k)} or \
        {k: entry[k] for k in ("labels_include_any", "labels_exclude_any", "labels_include_all") if entry.get(k)}
    f.update(labels)
    return f


def _payload_display_name(body: bytes):
    try:
        return plistlib.loads(body).get("PayloadDisplayName")
    except Exception:  # noqa: BLE001 - not parseable as a plist ($FLEET_VAR_ in a <data> element, say)
        m = re.findall(rb"<key>PayloadDisplayName</key>\s*<string>(.*?)</string>", body, re.S)
        return m[-1].decode(errors="replace") if m else None


def unescape_env(text: str) -> str:
    """What fleetctl sees after env expansion, for files that only use escapes (no real $VARs).

    Mirrors fleetctl's escapeString: an odd run of k backslashes before "$" becomes (k-1)/2
    backslashes and a literal "$"; an even run is left alone.
    """
    return re.sub(r"(\\+)\$", lambda m: m.group(0) if len(m.group(1)) % 2 == 0 else "\\" * ((len(m.group(1)) - 1) // 2) + "$", text)


class Checker:
    def __init__(self, root: Path, plan: dict, log=print):
        self.root = Path(root)
        self.plan = plan
        self.log = log
        self.results: dict = {"static": [], "references": [], "reconcile": [], "dry_run": {}}
        self.backend = yaml_backend()

    def fail(self, section, message):
        self.results[section].append({"level": "FAIL", "message": message})

    def warn(self, section, message):
        self.results[section].append({"level": "WARN", "message": message})

    # ------------------------------------------------------------ static

    def run_validator(self, offline: bool):
        script = self.root / ".agents/skills/fleet-gitops/scripts/validate.py"
        if not script.exists():
            self.warn("static", "the scaffold has no fleet-gitops validator; skipped")
            return
        args = [sys.executable, str(script), str(self.root)] + (["--offline"] if offline else [])
        r = subprocess.run(args, capture_output=True, text=True, timeout=900)
        for ln in r.stdout.splitlines():
            m = re.match(r"\s*(FAIL|WARN)\b[:\s]*(.*)", ln)
            if m:
                self.results["static"].append({"level": m.group(1), "message": m.group(2)})
        self.results["validator_exit"] = r.returncode

    def load_repo(self):
        """Parse every YAML file fleetctl would read, starting from default.yml and fleets/*.yml."""
        self.docs: dict = {}
        if not self.backend:
            self.warn("references", "no YAML parser (install PyYAML or Ruby); YAML parsing, reference, and reconcile checks skipped; the dry run still parses everything")
            return False
        tops = ["default.yml"] + sorted(str(p.relative_to(self.root)).replace(os.sep, "/") for p in (self.root / "fleets").glob("*.yml"))
        for rel in sorted(str(p.relative_to(self.root)).replace(os.sep, "/") for p in self.root.rglob("*.y*ml")):
            if rel.startswith(IGNORED_ROOTS) or rel in IGNORED_FILES or rel == ".gitlab-ci.yml":
                continue
            try:
                self.docs[rel] = load_yaml(unescape_env((self.root / rel).read_text(encoding="utf-8")), self.backend)
            except Exception as e:  # noqa: BLE001 - either parser's error
                self.fail("references", f"{rel}: YAML doesn't parse: {e}")
        self.tops = [t for t in tops if t in self.docs]
        return True

    def _refs(self, rel: str, node, out: list, key_path=()):
        base = os.path.dirname(rel)
        if isinstance(node, dict):
            for k, v in node.items():
                if k in PATH_KEYS and isinstance(v, str) and v:
                    out.append((k, os.path.normpath(os.path.join(base, v)).replace(os.sep, "/"), key_path + (k,)))
                else:
                    self._refs(rel, v, out, key_path + (k,))
        elif isinstance(node, list):
            for v in node:
                self._refs(rel, v, out, key_path)

    def references(self):
        referenced, missing = set(), []
        queue = list(self.tops)
        seen = set()
        self.edges: dict = {}
        while queue:
            rel = queue.pop()
            if rel in seen:
                continue
            seen.add(rel)
            refs: list = []
            self._refs(rel, self.docs.get(rel), refs)
            self.edges[rel] = refs
            for key, target, kp in refs:
                if key == "paths":
                    matches = [str(Path(m).relative_to(self.root)).replace(os.sep, "/")
                               for m in globlib.glob(str(self.root / target)) if Path(m).is_file()]
                    for m in matches:
                        referenced.add(m)
                        if m.endswith((".yml", ".yaml")):
                            queue.append(m)
                    continue
                if not (self.root / target).exists():
                    missing.append(f"{rel}: {'.'.join(map(str, kp))} points at {target}, which doesn't exist")
                    continue
                referenced.add(target)
                if target.endswith((".yml", ".yaml")):
                    queue.append(target)
        for m in missing:
            self.fail("references", m)
        orphans = []
        for p in self.root.rglob("*"):
            rel = str(p.relative_to(self.root)).replace(os.sep, "/")
            if not p.is_file() or rel.startswith(IGNORED_ROOTS) or rel in IGNORED_FILES or p.name == ".gitkeep" \
                    or rel in self.tops or rel in referenced:
                continue
            orphans.append(rel)
        for o in orphans:
            self.fail("references", f"{o} isn't referenced by any YAML file, so it does nothing")

    # ------------------------------------------------------------ reconcile

    def _items(self, top: str, section: list):
        """Resolved items under a section of a top-level file, following path/paths into item files."""
        node = self.docs.get(top) or {}
        for k in section:
            node = (node or {}).get(k) if isinstance(node, dict) else None
        out = []
        base = os.path.dirname(top)
        for entry in node or []:
            if not isinstance(entry, dict):
                continue
            if "paths" in entry:
                pattern = os.path.normpath(os.path.join(base, entry["paths"]))
                for m in sorted(globlib.glob(str(self.root / pattern))):
                    out.append((str(Path(m).relative_to(self.root)).replace(os.sep, "/"), entry))
            elif "path" in entry:
                out.append((os.path.normpath(os.path.join(base, entry["path"])).replace(os.sep, "/"), entry))
            else:
                out.append((None, entry))
        return out

    def _yaml_items(self, top: str, section: list):
        """Item dicts under a section: entries of referenced item files, or the entries themselves when inline."""
        out = []
        for path, entry in self._items(top, section):
            if path and path.endswith((".yml", ".yaml")):
                out.extend(item for item in self.docs.get(path) or [] if isinstance(item, dict))
            elif path is None:
                out.append(entry)
        return out

    def _bytes(self, path: str, expanded: bool) -> bytes:
        """A file's bytes as fleetctl sends them: env-unescaped for the file kinds it expands."""
        b = (self.root / path).read_bytes()
        if expanded:
            try:
                return unescape_env(b.decode("utf-8")).encode("utf-8")
            except UnicodeDecodeError:
                return b
        return b

    def _found(self, top: str) -> dict:
        """What the repo defines for a top-level file, keyed the same way as the plan's inventory."""
        base = os.path.dirname(top)
        found = {k: [] for k in KIND_NAMES}
        for item in self._yaml_items(top, ["policies"]):
            f = dict(item)
            f["type"] = item.get("type") or "dynamic"
            rs = item.get("run_script")
            f["run_script"] = os.path.basename(rs["path"]) if isinstance(rs, dict) and rs.get("path") else None
            found["policies"].append([item.get("name"), f])
        for kind in ("reports", "labels"):
            found[kind] = [[item.get("name"), item] for item in self._yaml_items(top, [kind])]
        for path, _ in self._items(top, ["controls", "scripts"]):
            if path:
                found["scripts"].append([os.path.basename(path), {"sha256": _sha(self._bytes(path, False))}])
        for section in (["controls", "apple_settings", "configuration_profiles"], ["controls", "windows_settings", "configuration_profiles"],
                        ["controls", "android_settings", "configuration_profiles"]):
            for path, entry in self._items(top, section):
                if not path:
                    continue
                body = self._bytes(path, True)
                name = entry.get("name")
                if not name and path.endswith(".mobileconfig"):
                    name = _payload_display_name(body)
                name = name or os.path.splitext(os.path.basename(path))[0]
                f = {k: entry.get(k) for k in ("self_service", "hidden", "description", "labels_include_all", "labels_include_any", "labels_exclude_any")}
                f["sha256"] = _sha(body)
                if entry.get("activation"):
                    f["activation_sha256"] = _sha(self._bytes(os.path.normpath(os.path.join(base, entry["activation"])), False))
                found["profiles"].append([name, f])
        for path, _ in self._items(top, ["controls", "apple_settings", "assets"]):
            if path:
                found["assets"].append([os.path.splitext(os.path.basename(path))[0], {"sha256": _sha(self._bytes(path, True))}])
        sw = (self.docs.get(top) or {}).get("software") or {}
        for entry in sw.get("packages") or []:
            if not isinstance(entry, dict):
                continue
            path = os.path.normpath(os.path.join(base, entry["path"])).replace(os.sep, "/") if entry.get("path") else None
            if path and path.endswith((".yml", ".yaml")):
                for item in self.docs.get(path) or []:
                    found["software"].append([f"package:{item.get('hash_sha256')}", _package_settings(item, entry)])
            elif path:
                found["software"].append([f"package:{_sha(self._bytes(path, False))}", _package_settings({}, entry)])
            else:
                found["software"].append([f"package:{entry.get('hash_sha256')}", _package_settings({}, entry)])
        for item in sw.get("fleet_maintained_apps") or []:
            if isinstance(item, dict):
                found["software"].append([f"fma:{item.get('slug')}", item])
        for item in sw.get("app_store_apps") or []:
            if isinstance(item, dict):
                f = {} if item.get("versions") else item
                found["software"].append([f"app:{item.get('app_store_id')}:{item.get('platform', '')}", f])
        return found

    def reconcile(self):
        """Compare the repo with Fleet item by item: presence, then each field Fleet reported."""
        inventory = self.plan.get("inventory") or {}
        for top, kinds in inventory.items():
            if top not in self.docs:
                if any(kinds.values()):
                    self.fail("reconcile", f"{top} is missing, but Fleet has items for it")
                continue
            found = self._found(top)
            for kind, want_list in kinds.items():
                if kind not in found:
                    self.fail("reconcile", f"{top}: the check doesn't know how to compare {kind!r}; treat it as unverified")
                    continue
                what = KIND_NAMES[kind]
                want = dict(map(tuple, want_list))
                got_list = found[kind]
                got = dict(map(tuple, got_list))
                keys = [k for k, _ in got_list]
                for n in sorted(set(want) - set(got), key=str):
                    self.fail("reconcile", f"{top}: {what} {n!r} is in Fleet but not in the repo (applying would delete it)")
                for n in sorted(set(got) - set(want), key=str):
                    self.fail("reconcile", f"{top}: {what} {n!r} is in the repo but not in Fleet (applying would create it)")
                for n in sorted({k for k in keys if keys.count(k) > 1}, key=str):
                    self.fail("reconcile", f"{top}: {what} {n!r} is defined more than once")
                for n in sorted(set(want) & set(got), key=str):
                    for field, fleet_value in want[n].items():
                        if _norm(fleet_value) != _norm(got[n].get(field)):
                            self.fail("reconcile", f"{top}: {what} {n!r}: {field} is {_short(got[n].get(field))} in the repo "
                                                   f"but {_short(fleet_value)} in Fleet (applying would change it)")

    # ------------------------------------------------------------ dry run

    def dry_run_env(self, api: API) -> tuple[dict, list]:
        env = dict(os.environ)
        notes = []
        cache: dict = {}

        def get(path, params=None):
            key = (path, json.dumps(params or {}, sort_keys=True))
            if key not in cache:
                cache[key] = api.json(path, params)
            return cache[key]

        def walk(d, pointer):
            for p in pointer:
                d = d[p] if isinstance(d, dict) else d[int(p)]
            return d

        for e in self.plan["env"]:
            name, src = e["name"], e["source"]
            t = src["type"]
            try:
                if t == "fleet_url":
                    val = get(f"{F}/config")["server_settings"]["server_url"]
                elif t == "enroll_secret":
                    if src["team"] is None:
                        secrets = (get(f"{F}/spec/enroll_secret").get("spec") or {}).get("secrets") or []
                    else:
                        d = get(f"{F}/fleets/{src['team']}")
                        secrets = (d.get("fleet") or d.get("team") or {}).get("secrets") or []
                    val = secrets[src["index"]]["secret"]
                elif t == "config":
                    val = walk(get(f"{F}/config"), src["pointer"])
                elif t == "team_config":
                    d = get(f"{F}/fleets/{src['team']}")
                    val = walk(d.get("fleet") or d.get("team"), src["pointer"])
                elif t == "ca":
                    cas = get(f"{F}/spec/certificate_authorities", {"include_secrets": "true"})["certificate_authorities"]
                    item = cas[src["kind"]]
                    item = item[src["index"]] if isinstance(item, list) else item
                    val = item[src["field"]]
                elif t == "mask":
                    val = os.environ.get(name) or MASK
                    if not os.environ.get(name):
                        notes.append(f"{name}: Fleet only returns this value masked; the dry run used Fleet's mask, which Fleet treats as unchanged")
                else:
                    val = os.environ.get(name)
                    if not val:
                        val = PLACEHOLDER
                        notes.append(f"{name}: not set in this shell; the dry run used a placeholder, so files using it weren't checked with the real value")
            except (KeyError, IndexError, TypeError, FleetctlError) as ex:
                val = PLACEHOLDER
                notes.append(f"{name}: couldn't read it from Fleet ({ex}); the dry run used a placeholder")
            if src.get("indent") and isinstance(val, str) and "\n" in val:
                # Same trick as the commented-out lines in .github/fleet-gitops/gitops.sh: keep a multi-line value inside its YAML key.
                lines = val.split("\n")
                val = "\n".join([lines[0]] + [" " * src["indent"] + ln for ln in lines[1:]])
            env[name] = "" if val is None else str(val)
        return env, notes

    def dry_run(self, api: API, fleetctl):
        env, notes = self.dry_run_env(api)
        files = ["default.yml"] + sorted("fleets/" + p.name for p in (self.root / "fleets").glob("*.yml"))
        r = fleetctl.gitops_dry_run(files, cwd=str(self.root), env=env)
        out = (r.stdout.decode(errors="replace") + r.stderr.decode(errors="replace"))
        changes = [ln for ln in out.splitlines()
                   if ln.startswith("[-]") or "would've deleted" in ln or "would've created" in ln or "would've moved" in ln]
        self.results["dry_run"] = {
            "command": "fleetctl gitops " + " ".join(f"-f {f}" for f in files) + " --delete-other-fleets --dry-run",
            "exit_code": r.returncode,
            "changes": changes,
            "notes": notes,
            "output": out,
        }
        if r.returncode:
            self.fail("reconcile", "fleetctl gitops --dry-run failed; see the dry-run output")
        for ln in changes:
            self.fail("reconcile", f"dry run reports a change: {ln.strip()}")

    def summary(self) -> dict:
        # The fleet-gitops validator also flags content problems that came from Fleet as-is
        # (an unknown osquery table, say); they're reported but don't fail the export.
        levels = [r["level"] for k in ("references", "reconcile") for r in self.results[k]]
        self.results["passed"] = "FAIL" not in levels
        self.results["yaml_backend"] = self.backend
        return self.results
