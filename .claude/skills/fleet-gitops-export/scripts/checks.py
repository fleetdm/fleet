"""Validate an exported repo: YAML parses, references resolve, nothing is orphaned,
the repo defines exactly what Fleet has, and a fleetctl dry run is clean.

`fleetctl gitops --dry-run` prints "would've applied N" for most sections
without diffing them, and only reports deletions for policies, reports,
labels, software packages, custom host vitals, Android certificates, and
fleets. The reconcile check fills that gap: it resolves every path and glob
on disk and compares the result with what Fleet has, item by item and field
by field (content hashes for scripts, profiles, and DDM assets), along with
the settings and controls Fleet reports that aren't held in variables.
"""

from __future__ import annotations

import fnmatch
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
from collect import MIN_SECRET_LEN
from planner import ANY_OF, CATALOG_CATEGORIES, TOP_LEVEL, VARIABLE

F = "/api/latest/fleet"
PLACEHOLDER = "fleet-gitops-export-dry-run-placeholder"
MASK = "********"
UNCHECKED = "(not compared)"
MISSING = "(missing file)"

PATH_KEYS = {"path", "paths", "package_path", "activation", "apple_setup_assistant", "macos_script",
             "end_user_license_agreement", "windows_eula", "org_logo_path_light_mode", "org_logo_path_dark_mode"}
IGNORED_ROOTS = (".github/", ".agents/", ".claude/", ".git/")
JUNK_FILES = {".gitkeep", ".DS_Store", "Thumbs.db", "desktop.ini"}  # never referenced, and that's fine
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
              "profiles": "profile", "assets": "DDM asset", "software": "software", "settings": "settings",
              "certificates": "Android certificate"}
LABEL_KEYS = ("labels_include_any", "labels_exclude_any", "labels_include_all")
APP_FIELDS = ("auto_update_enabled", "auto_update_window_start", "auto_update_window_end")
CERT_FIELDS = ("certificate_authority_name", "subject_name", "subject_alternative_name")
# Settings whose omitted value isn't false (fleetctl fills these in on apply).
OMITTED = {("controls", "enable_profile_validation"): True,
           **{(s, k): True for s in ("org_settings.features", "settings.features") for k in ("enable_host_users", "enable_software_inventory")},
           **{(s, "historical_data"): {"uptime": True, "vulnerabilities": True} for s in ("org_settings.features", "settings.features")}}
# Compared exactly: an empty or false value here means something different from a missing one.
STRICT = {(TOP_LEVEL, "agent_options"), ("controls.windows_updates", "deadline_days"), ("controls.windows_updates", "grace_period_days")}
# Repo sections compared item by item (kinds), so their keys are covered.
ITEM_SECTIONS = {("policies",), ("reports",), ("labels",), ("controls", "scripts"), ("controls", "apple_settings", "assets"),
                 ("controls", "apple_settings", "configuration_profiles"), ("controls", "windows_settings", "configuration_profiles"),
                 ("controls", "android_settings", "configuration_profiles"), ("controls", "android_settings", "certificates"),
                 ("software", "packages"), ("software", "fleet_maintained_apps"), ("software", "app_store_apps")}
# Old names fleetctl still maps (pkg/spec/gitops_deprecations.go); the export writes the new ones.
DEPRECATED_PATHS = {"team_settings": "settings", "queries": "reports", "controls.macos_settings": "controls.apple_settings",
                    "controls.macos_setup": "controls.setup_experience",
                    "controls.apple_settings.custom_settings": "controls.apple_settings.configuration_profiles",
                    "controls.windows_settings.custom_settings": "controls.windows_settings.configuration_profiles",
                    "controls.android_settings.custom_settings": "controls.android_settings.configuration_profiles",
                    "controls.setup_experience.bootstrap_package": "controls.setup_experience.macos_bootstrap_package",
                    "controls.setup_experience.macos_setup_assistant": "controls.setup_experience.apple_setup_assistant",
                    "controls.setup_experience.enable_release_device_manually": "controls.setup_experience.apple_enable_release_device_manually",
                    "controls.setup_experience.script": "controls.setup_experience.macos_script",
                    "controls.setup_experience.manual_agent_install": "controls.setup_experience.macos_manual_agent_install",
                    "controls.setup_experience.enable_managed_local_account": "controls.setup_experience.enable_create_local_admin_account",
                    "org_settings.server_settings.live_query_disabled": "org_settings.server_settings.live_reporting_disabled",
                    "org_settings.server_settings.query_reports_disabled": "org_settings.server_settings.discard_reports_data",
                    "org_settings.server_settings.query_report_cap": "org_settings.server_settings.report_cap",
                    "org_settings.org_info.org_logo_url": "org_settings.org_info.org_logo_url_dark_mode",
                    "org_settings.org_info.org_logo_url_light_background": "org_settings.org_info.org_logo_url_light_mode",
                    "org_settings.mdm.apple_business_manager": "org_settings.mdm.apple_business"}
# Settings where the repo's other keys are read too, so a key only the repo has (a kind, a Premium-only flag) is an addition.
KEYED_SECTIONS = ("org_settings.integrations", "org_settings.certificate_authorities", "org_settings.sso_settings")


def _with_defaults(default, v):
    """A repo value with what fleetctl fills in when it's left out (nested for historical_data)."""
    if default is None:
        return v
    if isinstance(default, dict):
        return {**default, **{k: x for k, x in (v if isinstance(v, dict) else {}).items() if x is not None}}
    return default if v is None else v


def _strict(v):
    """Comparable form for STRICT settings: only None counts as absent; false, 0, and empty values stay."""
    if isinstance(v, dict):
        return {k: _strict(x) for k, x in v.items() if x is not None} or None
    if isinstance(v, list):
        return sorted((_strict(x) for x in v), key=lambda x: json.dumps(x, sort_keys=True))
    return v


def _has_value(v) -> bool:
    """Would applying this set something? false and 0 do; None, "", and empty containers don't."""
    if isinstance(v, dict):
        return any(_has_value(x) for x in v.values())
    if isinstance(v, list):
        return bool(v)
    return v is not None and v != ""


def _dotted(path) -> str:
    return "".join(f"[{p}]" if isinstance(p, int) else ("." if i else "") + str(p) for i, p in enumerate(path))


def _hashable(v):
    """An item key as reconcile indexes it; a YAML list or mapping where a name belongs still gets compared."""
    return v if isinstance(v, (str, int, float, bool, type(None))) else json.dumps(v, sort_keys=True, default=str)


def _as_list(v) -> list:
    """A YAML item file's entries: fleetctl also accepts a single mapping."""
    return v if isinstance(v, list) else [v] if isinstance(v, dict) else []


def _gitignore(root: Path):
    """A matcher for the repo's .gitignore (plain patterns; enough for what the scaffold and users add)."""
    try:
        lines = (root / ".gitignore").read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeDecodeError):
        lines = []
    pats = [ln.strip() for ln in lines if ln.strip() and not ln.lstrip().startswith(("#", "!"))]

    def ignored(rel: str) -> bool:
        parts = rel.split("/")
        for p in pats:
            p = p.rstrip("/")
            if "/" in p.lstrip("/"):
                if fnmatch.fnmatch(rel, p.lstrip("/")) or fnmatch.fnmatch(rel, p.lstrip("/") + "/*"):
                    return True
            elif any(fnmatch.fnmatch(part, p) for part in parts):
                return True
        return False
    return ignored


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


def _same(fleet_value, repo_value) -> bool:
    if isinstance(fleet_value, dict) and set(fleet_value) == {ANY_OF}:
        return any(_norm(v) == _norm(repo_value) for v in fleet_value[ANY_OF])
    return _norm(fleet_value) == _norm(repo_value)


def _short(v) -> str:
    if isinstance(v, dict) and set(v) == {ANY_OF}:
        return "one of " + _short(v[ANY_OF])
    text = json.dumps(v, ensure_ascii=False, default=str)
    return text if len(text) <= 80 else text[:77] + "..."


def _walk(node, path: list):
    for k in path:
        node = node.get(k) if isinstance(node, dict) else None
    return node


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


def read_env_file(path: Path) -> dict:
    """NAME=value lines from a .env file (optional `export`, quotes, multi-line quoted values).

    Empty values are dropped, so an untouched copy of .env.example sets nothing.
    """
    values, name, quote, buf = {}, None, None, []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        if quote:
            if line.endswith(quote):
                values[name] = "\n".join(buf + [line[:-1]])
                quote = None
            else:
                buf.append(line)
            continue
        m = re.match(r"\s*(?:export\s+)?([A-Za-z_]\w*)=(.*)$", line)
        if not m:
            continue
        name, val = m.group(1), m.group(2).strip()
        if val[:1] in ("'", '"'):
            if len(val) > 1 and val.endswith(val[0]):
                values[name] = val[1:-1]
            else:
                quote, buf = val[0], [val[1:]]
            continue
        values[name] = re.sub(r"\s+#.*$", "", val)
    return {k: v for k, v in values.items() if v}


class Checker:
    def __init__(self, root: Path, plan: dict, log=print):
        self.root = Path(root)
        self.plan = plan
        self.log = log
        self.results: dict = {"static": [], "references": [], "reconcile": [], "dry_run": {}}
        self.backend = yaml_backend()
        # Files where the export wrote $VARIABLEs in place of values Fleet stored (enroll secrets in profiles).
        self.env_refs = {f["path"]: f["env_refs"] for f in (plan.get("files") or {}).values() if f.get("env_refs")}
        self.env_sources = {e["name"]: e["source"] for e in plan.get("env") or []}
        self.env_values: dict | None = None
        self.env_notes: list = []
        self.compared = False  # set once reconcile or the dry run compares the repo with Fleet
        self.secret_values: dict = {}

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
        try:
            r = subprocess.run(args, capture_output=True, text=True, timeout=900)
        except (OSError, subprocess.SubprocessError) as e:
            self.warn("static", f"the fleet-gitops validator didn't finish ({e}); its checks were skipped")
            return
        for ln in r.stdout.splitlines():
            m = re.match(r"\s*(FAIL|WARN)\b[:\s]*(.*)", ln)
            if m:
                self.results["static"].append({"level": m.group(1), "message": m.group(2)})
        self.results["validator_exit"] = r.returncode
        if r.returncode and not self.results["static"]:
            last = (r.stderr.strip().splitlines() or ["no output"])[-1]
            self.warn("static", f"the fleet-gitops validator exited {r.returncode} without reporting findings ({last}); its checks didn't run")

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
        for t in tops:
            if t in self.docs and not isinstance(self.docs[t], dict):
                self.fail("references", f"{t} isn't a YAML mapping, so fleetctl can't read it")
                self.docs[t] = {}
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
                if not self._inside(target):
                    missing.append(f"{rel}: {'.'.join(map(str, kp))} points at {target}, which is outside the repo")
                    continue
                if key == "paths":
                    for m in self._glob(target):
                        referenced.add(m)
                        if m.endswith((".yml", ".yaml")):
                            queue.append(m)
                    continue
                if not (self.root / target).is_file():
                    what = "isn't a file" if (self.root / target).exists() else "doesn't exist"
                    missing.append(f"{rel}: {'.'.join(map(str, kp))} points at {target}, which {what}")
                    continue
                referenced.add(target)
                if target.endswith((".yml", ".yaml")):
                    queue.append(target)
        for m in missing:
            self.fail("references", m)
        orphans = []
        ignored = _gitignore(self.root)
        for p in self.root.rglob("*"):
            rel = str(p.relative_to(self.root)).replace(os.sep, "/")
            if not p.is_file() or rel.startswith(IGNORED_ROOTS) or rel in IGNORED_FILES or p.name in JUNK_FILES \
                    or rel in self.tops or rel in referenced or ignored(rel):
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
        for entry in node if isinstance(node, list) else []:
            if not isinstance(entry, dict):
                continue
            if isinstance(entry.get("paths"), str):
                pattern = os.path.normpath(os.path.join(base, entry["paths"])).replace(os.sep, "/")
                out.extend((m, entry) for m in (self._glob(pattern) if self._inside(pattern) else []))
            elif isinstance(entry.get("path"), str):
                p = os.path.normpath(os.path.join(base, entry["path"])).replace(os.sep, "/")
                out.append((p if self._inside(p) else MISSING, entry))
            else:
                out.append((None, entry))
        return out

    def _inside(self, rel: str) -> bool:
        return not (os.path.isabs(rel) or rel == ".." or rel.startswith("../"))

    def _glob(self, pattern: str) -> list:
        """Repo files a `paths` glob matches, as repo-relative paths."""
        root = self.root.resolve()
        out = []
        for m in sorted(globlib.glob(str(self.root / pattern))):
            p = Path(m)
            if p.is_file() and p.resolve().is_relative_to(root):
                out.append(str(p.relative_to(self.root)).replace(os.sep, "/"))
        return out

    def _yaml_items(self, top: str, section: list):
        """(folder paths in it are relative to, item dict) under a section: entries of referenced item files,
        or the entries themselves when inline."""
        out = []
        for path, entry in self._items(top, section):
            if path and path.endswith((".yml", ".yaml")):
                out.extend((os.path.dirname(path), item) for item in _as_list(self.docs.get(path)) if isinstance(item, dict))
            elif path is None:
                out.append((os.path.dirname(top), entry))
        return out

    def _strip_env(self, v):
        """A setting without the values the repo holds as variables (tokens, URLs), which Fleet's side leaves out too.

        A certificate authority secret's variable becomes {VARIABLE: where it was planned for}, as Fleet's side has it.
        """
        if isinstance(v, dict):
            out = {}
            for k, x in v.items():
                src = self._ref_source(x) if isinstance(x, str) else None
                if src is None:
                    out[k] = self._strip_env(x)
                elif "kind" in src:
                    out[k] = {VARIABLE: f"{src['kind']}[{src['index']}].{src['field']}"}
            return out
        if isinstance(v, list):
            return [self._strip_env(x) for x in v if not (isinstance(x, str) and self._ref_source(x))]
        return v

    def _ref_source(self, x: str):
        """The planned source of a value that's just a $VARIABLE reference, else None."""
        m = re.fullmatch(r"\$\{?(\w+)\}?", x)
        return self.env_sources.get(m.group(1)) if m else None

    def _ref(self, base: str, ref) -> str | None:
        """The repo path a `{path: ...}` value points at."""
        if isinstance(ref, dict) and ref.get("path"):
            return os.path.normpath(os.path.join(base, ref["path"])).replace(os.sep, "/")
        return None

    def _script_sha(self, base: str, ref):
        path = self._ref(base, ref)
        return self._hash(path) if path else None

    def _query(self, base: str, ref):
        items = _as_list(self.docs.get(self._ref(base, ref) or ""))
        return items[0].get("query") if items and isinstance(items[0], dict) else None

    def _app_config(self, base: str, ref, platform: str):
        path = self._ref(base, ref)
        if not path:
            return None
        if not (self.root / path).is_file():
            return MISSING
        if platform in ("ios", "ipados"):
            return _sha(self._bytes(path, False))
        body = self._bytes(path, False)
        try:
            return json.loads(body)
        except ValueError:
            return body.decode(errors="replace")

    def _software_fields(self, scripts: dict, base: str, entry: dict, entry_base: str | None = None) -> dict:
        """A software item's settings: its own, else the fleet-level entry's (as fleetctl merges them),
        with its scripts and query as reconcile compares them (scripts by hash). Paths in `scripts` are
        relative to `base`, paths in `entry` (in the fleet file) to `entry_base`."""
        entry_base = base if entry_base is None else entry_base
        f = {k: scripts.get(k) or entry.get(k) for k in ("self_service", "categories", "display_name")}
        labels = {k: scripts[k] for k in LABEL_KEYS if scripts.get(k)} or {k: entry[k] for k in LABEL_KEYS if entry.get(k)}
        f.update({k: labels.get(k) for k in LABEL_KEYS})
        for key in ("install_script", "uninstall_script", "post_install_script"):
            f[key] = self._script_sha(base, scripts.get(key))
        f["pre_install_query"] = self._query(base, scripts.get("pre_install_query"))
        for key in ("setup_experience", "setup_experience_platform"):
            f[key] = scripts.get(key) or entry.get(key)
        f["icon"] = self._script_sha(base, scripts["icon"]) if scripts.get("icon") else self._script_sha(entry_base, entry.get("icon"))
        if entry.get("configuration"):
            f["configuration"] = self._app_config(entry_base, entry["configuration"], "ios")
        return f

    def _install_software(self, base: str, inst):
        """What a policy's install_software names, resolved the way the plan's inventory keys it."""
        if not isinstance(inst, dict):
            return inst
        if inst.get("fleet_maintained_app_slug"):
            return f"fma:{inst['fleet_maintained_app_slug']}"
        if inst.get("app_store_id"):
            return f"app:{inst['app_store_id']}"
        if inst.get("package_path"):
            items = _as_list(self.docs.get(self._ref(base, {"path": inst["package_path"]})))
            first = items[0] if items and isinstance(items[0], dict) else {}
            return f"package:{first.get('url') or first.get('hash_sha256')}"
        return f"package:{inst.get('hash_sha256')}"

    def _bytes(self, path: str, expanded: bool) -> bytes:
        """A file's bytes as fleetctl sends them: env-unescaped for the file kinds it expands."""
        b = (self.root / path).read_bytes()
        if expanded:
            try:
                return unescape_env(b.decode("utf-8")).encode("utf-8")
            except UnicodeDecodeError:
                return b
        return b

    def _hash(self, path: str, expanded: bool = False) -> str:
        """A file's hash as fleetctl sends it (`expanded` for profiles and DDM assets).

        MISSING when the path isn't a file, which references() already fails, so the check still finishes.
        """
        if not (self.root / path).is_file():
            return MISSING
        body = self._expanded_bytes(path) if expanded else self._bytes(path, False)
        return UNCHECKED if body is None else _sha(body)

    def _expanded_bytes(self, path: str) -> bytes | None:
        """A profile or asset as fleetctl sends it, with the variables the export wrote filled in.

        None when it has such variables and their values weren't fetched (validate --no-dry-run).
        """
        names = self.env_refs.get(path)
        if not names:
            return self._bytes(path, True)
        if self.env_values is None:
            return None
        text = (self.root / path).read_text(encoding="utf-8")
        for n in names:
            value = self.env_values.get(n, "")
            text = re.sub(r"(?<!\\)\$(?:\{" + n + r"\}|" + n + r"(?![A-Za-z0-9_]))", lambda _m: value, text)
        return unescape_env(text).encode("utf-8")

    def _found(self, top: str, settings: list) -> dict:
        """What the repo defines for a top-level file, keyed the same way as the plan's inventory.

        `settings` is the inventory's settings for this file: which fields to read where.
        """
        base = os.path.dirname(top)
        doc = self.docs.get(top) or {}
        found = {k: [] for k in KIND_NAMES}
        for item_base, item in self._yaml_items(top, ["policies"]):
            f = dict(item)
            f["type"] = item.get("type") or "dynamic"
            rs = item.get("run_script")
            f["run_script"] = os.path.basename(rs["path"]) if isinstance(rs, dict) and rs.get("path") else None
            f["install_software"] = self._install_software(item_base, item.get("install_software"))
            found["policies"].append([item.get("name"), f])
        for kind in ("reports", "labels"):
            found[kind] = [[item.get("name"), item] for _, item in self._yaml_items(top, [kind])]
        for section, fields in settings:
            node = doc if section == TOP_LEVEL else _walk(doc, section.split("."))
            values = {}
            for k in fields:
                v = _walk(node, [k])
                if k == "secrets" and section in ("org_settings", "settings"):
                    values[k] = len(v) if isinstance(v, list) else 0  # which variable each holds is checked by placement
                elif k in PATH_KEYS and isinstance(v, str):
                    values[k] = self._file_value(k, os.path.normpath(os.path.join(base, v)).replace(os.sep, "/"))
                else:
                    values[k] = _with_defaults(OMITTED.get((section, k)), self._files_in(self._strip_env(v), base))
            if section in KEYED_SECTIONS and isinstance(node, dict):
                values.update({k: v for k, v in self._strip_env(node).items() if k not in fields})
            found["settings"].append([section, values])
        for path, _ in self._items(top, ["controls", "scripts"]):
            if path:
                found["scripts"].append([os.path.basename(path), {"sha256": self._hash(path)}])
        for section in (["controls", "apple_settings", "configuration_profiles"], ["controls", "windows_settings", "configuration_profiles"],
                        ["controls", "android_settings", "configuration_profiles"]):
            for path, entry in self._items(top, section):
                if not path:
                    continue
                name = entry.get("name")
                if not name and path.endswith(".mobileconfig") and (self.root / path).is_file():
                    name = _payload_display_name(self._bytes(path, True))
                name = name or os.path.splitext(os.path.basename(path))[0]
                f = {k: entry.get(k) for k in ("self_service", "hidden", "description", "labels_include_all", "labels_include_any", "labels_exclude_any")}
                f["sha256"] = self._hash(path, expanded=True)
                act = entry.get("activation")
                f["activation_sha256"] = self._hash(os.path.normpath(os.path.join(base, act)).replace(os.sep, "/")) if isinstance(act, str) else None
                found["profiles"].append([name, f])
        for path, _ in self._items(top, ["controls", "apple_settings", "assets"]):
            if path:
                found["assets"].append([os.path.splitext(os.path.basename(path))[0], {"sha256": self._hash(path, expanded=True)}])
        sw = doc.get("software") if isinstance(doc.get("software"), dict) else {}
        for entry in _as_list(sw.get("packages")):
            if not isinstance(entry, dict):
                continue
            path = self._ref(base, entry)
            if path and path.endswith((".yml", ".yaml")):
                for item in (i for i in _as_list(self.docs.get(path)) if isinstance(i, dict)):
                    found["software"].append([f"package:{item.get('url') or item.get('hash_sha256')}",
                                              self._software_fields(item, os.path.dirname(path), entry, base)])
            elif path:
                found["software"].append([f"package:{self._hash(path)}", self._software_fields(entry, base, entry)])
            else:
                found["software"].append([f"package:{entry.get('hash_sha256')}", self._software_fields(entry, base, entry)])
        for item in _as_list(sw.get("fleet_maintained_apps")):
            if isinstance(item, dict):
                f = self._software_fields(item, base, item)
                f["version"] = item.get("version")
                if "categories" not in item:
                    f["categories"] = CATALOG_CATEGORIES  # left out, the FMA gets the catalog's categories
                found["software"].append([f"fma:{item.get('slug')}", f])
        for item in _as_list(sw.get("app_store_apps")):
            if not isinstance(item, dict):
                continue
            ident, platform = f"app:{item.get('app_store_id')}:{item.get('platform', '')}", item.get("platform", "")
            top = {"display_name": item.get("display_name"), "setup_experience": item.get("setup_experience"),
                   "icon": self._script_sha(base, item.get("icon")), "location": item.get("location")}
            if item.get("versions"):
                found["software"].append([ident, top])
            versions = [(f"{ident}:{v.get('name')}", v) for v in item.get("versions") or [] if isinstance(v, dict)] or [(ident, item)]
            for key, v in versions:
                f = {k: v.get(k) for k in ("self_service", "categories", "display_name") + LABEL_KEYS + APP_FIELDS}
                f["configuration"] = self._app_config(base, v.get("configuration"), platform)
                if v is item:
                    f.update(top)
                found["software"].append([key, f])
        for item in _as_list(_walk(doc, ["controls", "android_settings", "certificates"])):
            if isinstance(item, dict):
                found["certificates"].append([item.get("name"), {k: item.get(k) for k in CERT_FIELDS}])
        return found

    def _file_value(self, key: str, path: str):
        """A file a setting points at, as the inventory has it: the ADE profile parsed, anything else hashed."""
        if key != "apple_setup_assistant" or not (self.root / path).is_file():
            return self._hash(path)
        try:
            return json.loads(self._bytes(path, False))
        except ValueError:
            return self._hash(path)

    def _files_in(self, v, base: str):
        """A setting with each `{path: ...}` it holds (YARA rules) replaced by that file's hash."""
        if isinstance(v, dict) and set(v) == {"path"} and isinstance(v["path"], str):
            return self._hash(os.path.normpath(os.path.join(base, v["path"])).replace(os.sep, "/"))
        if isinstance(v, list):
            return [self._files_in(x, base) for x in v]
        return v

    def reconcile(self):
        """Compare the repo with Fleet item by item: presence, then each field Fleet reported.

        Every kind is compared for every fleet file, so a file Fleet has nothing of a kind for
        still fails when the repo defines some.
        """
        inventory = self.plan.get("inventory") or {}
        exceptions = self.plan.get("exceptions") or {}
        planned = list(dict.fromkeys(f["file"] for f in self.plan.get("fleets") or []))
        for top in self.tops:
            if planned and top not in planned:
                self.fail("reconcile", f"{top} isn't a fleet in Fleet (applying would create it)")
        self.compared = True
        for top in planned + [t for t in inventory if t not in planned]:
            if top not in self.docs:
                if (self.root / top).exists():
                    self.fail("reconcile", f"{top} doesn't parse, so it wasn't compared (applying would fail)")
                else:
                    self.fail("reconcile", f"{top} is missing (applying would remove what Fleet has for it)")
                continue
            try:
                self._reconcile_file(top, inventory.get(top) or {}, exceptions)
            except Exception as e:  # noqa: BLE001 - YAML of an unexpected shape; the other files are still compared
                self.fail("reconcile", f"{top}: couldn't be compared ({type(e).__name__}: {e}); check its YAML")
        self.check_placements()

    def _reconcile_file(self, top: str, kinds: dict, exceptions: dict):
        found = self._found(top, kinds.get("settings") or [])
        managed = [k for k in KIND_NAMES if not exceptions.get(k)]
        for kind in managed + [k for k in kinds if k not in managed]:
            want_list = kinds.get(kind) or []
            if kind not in found:
                self.fail("reconcile", f"{top}: the check doesn't know how to compare {kind!r}; treat it as unverified")
                continue
            what = KIND_NAMES[kind]
            want = dict(map(tuple, want_list))
            got_list = [[_hashable(k), v] for k, v in found[kind]]
            got = dict(map(tuple, got_list))
            keys = [k for k, _ in got_list]
            for n in sorted(set(want) - set(got), key=str):
                self.fail("reconcile", f"{top}: {what} {n!r} is in Fleet but not in the repo (applying would delete it)")
            for n in sorted(set(got) - set(want), key=str):
                self.fail("reconcile", f"{top}: {what} {n!r} is in the repo but not in Fleet (applying would create it)")
            for n in sorted({k for k in keys if keys.count(k) > 1}, key=str):
                self.fail("reconcile", f"{top}: {what} {n!r} is defined more than once")
            for n in sorted(set(want) & set(got), key=str):
                # Settings and YAML items hold whatever keys the repo gives them; anything Fleet has no value for is an addition.
                # (An item Fleet lists without fields was left out for holding a secret; it isn't compared field by field.)
                extra = sorted(set(got[n]) - set(want[n]) - {"name"}) if kind == "settings" or (kind in ("policies", "reports", "labels") and want[n]) else []
                for field in extra:
                    if _has_value(got[n][field]) and _norm(got[n][field]) is not None:
                        self.fail("reconcile", f"{top}: {what} {n!r}: {field} is in the repo but not in Fleet (applying would create it)")
                for field, fleet_value in want[n].items():
                    if got[n].get(field) == UNCHECKED:
                        self.warn("reconcile", f"{top}: {what} {n!r}: {field} not compared; the file holds variables "
                                               "whose values validate only fetches when it runs the dry run")
                        continue
                    if kind == "settings" and (n, field) in STRICT:
                        if _strict(fleet_value) != _strict(got[n].get(field)):
                            self.fail("reconcile", f"{top}: {what} {n!r}: {self._difference(n, field, fleet_value, got[n].get(field))}")
                        continue
                    if not _same(fleet_value, got[n].get(field)):
                        if field.endswith("sha256"):
                            # The hashes aren't printed: Fleet's is of content that may hold an enroll secret.
                            part = field[:-len("sha256")].replace("_", " ")
                            if got[n].get(field) == MISSING:
                                self.fail("reconcile", f"{top}: {what} {n!r}: its {part}file is missing from the repo")
                            else:
                                self.fail("reconcile", f"{top}: {what} {n!r}: {part or 'content '}differs between the repo and Fleet "
                                                       "(applying would change it)")
                            continue
                        self.fail("reconcile", f"{top}: {what} {n!r}: {self._difference(n, field, fleet_value, got[n].get(field))}")

        self._coverage(top, kinds.get("settings") or [])

    @staticmethod
    def _difference(section, field: str, fleet_value, repo_value) -> str:
        plain = all(isinstance(v, (bool, int, float, type(None))) for v in (fleet_value, repo_value))
        if field == "agent_options" or (str(section).endswith(".features") and not plain):
            # Credentials there are variables; one written out in the repo mustn't end up in validation.json.
            return f"{field} differs between the repo and Fleet (applying would change it)"
        return f"{field} is {_short(repo_value)} in the repo but {_short(fleet_value)} in Fleet (applying would change it)"

    def _coverage(self, top: str, settings: list):
        """Fail on repo keys no comparison reads: applying would set them, and nothing here checks them."""
        covered = set(ITEM_SECTIONS)
        for section, fields in settings:
            base = () if section == TOP_LEVEL else tuple(section.split("."))
            if section in KEYED_SECTIONS:
                covered.add(base)
            else:
                covered.update(base + (f,) for f in fields)
        covered.update(self._placements().get(top, {}))
        if top != "default.yml":
            covered.add(("name",))
        containers = {p[:i] for p in covered for i in range(1, len(p))}

        def walk(node, path):
            for k, v in (node.items() if isinstance(node, dict) else []):
                p = path + (k,)
                if p in covered:
                    continue
                if p in containers and (isinstance(v, dict) or v is None):
                    walk(v, p)
                elif _has_value(v):
                    old = DEPRECATED_PATHS.get(_dotted(p))
                    hint = f"; it's the old name of {old}, which the export writes" if old else ""
                    self.fail("reconcile", f"{top}: {_dotted(p)} is in the repo but isn't compared with Fleet (applying may change Fleet){hint}")
        walk(self.docs.get(top), ())

    def _placements(self) -> dict:
        """{file: {key path: variable}} for each $VARIABLE the plan put in a YAML file."""
        if getattr(self, "_placed", None) is None:
            self._placed = {}

            def walk(node, path, into):
                if isinstance(node, dict):
                    if set(node) == {"$env"}:
                        into[path] = node["$env"]
                        return
                    for k, v in node.items():
                        walk(v, path + (k,), into)
                elif isinstance(node, list):
                    for i, v in enumerate(node):
                        walk(v, path + (i,), into)
            for f in self.plan.get("fleets") or []:
                walk(f.get("sections") or {}, (), self._placed.setdefault(f["file"], {}))
            for f in (self.plan.get("files") or {}).values():
                if "data" in f:
                    walk(f["data"], (), self._placed.setdefault(f["path"], {}))
        return self._placed

    def check_placements(self):
        """Each variable is where the plan put it, and nowhere else: a value written out in its place, or another
        variable, would apply something other than what Fleet has (or what the variable holds)."""
        planned = self._placements()
        for rel in sorted(set(planned) & set(self.docs)):
            got: dict = {}

            def walk(node, path):
                if isinstance(node, dict):
                    for k, v in node.items():
                        walk(v, path + (k,))
                elif isinstance(node, list):
                    for i, v in enumerate(node):
                        walk(v, path + (i,))
                elif isinstance(node, str):
                    m = re.fullmatch(r"\$\{?(\w+)\}?", node)
                    if m and m.group(1) in self.env_sources:
                        got[path] = m.group(1)
            walk(self.docs[rel], ())
            want = planned[rel]
            for p, name in sorted(want.items(), key=lambda kv: _dotted(kv[0])):
                if got.get(p) != name:
                    now = f"${got[p]}" if p in got else "no variable"
                    self.fail("reconcile", f"{rel}: {_dotted(p)} should be ${name}, but the repo has {now} there (applying would set something else)")
            for p, name in sorted(got.items(), key=lambda kv: _dotted(kv[0])):
                if p not in want:
                    self.fail("reconcile", f"{rel}: {_dotted(p)} uses ${name}, which the export didn't put there (applying would set that value)")

    # ------------------------------------------------------------ dry run

    def dry_run_env(self, api: API, env_file: Path | None = None) -> tuple[dict, list]:
        env = dict(os.environ)
        notes = []
        cache: dict = {}
        # Agents often run commands in a shell the user can't export into, so values only the
        # user has come from the repo's gitignored .env first.
        try:
            local = read_env_file(env_file) if env_file else {}
        except (OSError, UnicodeDecodeError) as ex:
            local = {}
            notes.append(f"{env_file}: couldn't read it ({ex}); the dry run used the environment only")

        def given(name):
            return local.get(name) or os.environ.get(name)

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
            from_user = False
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
                elif t == "api":
                    val = walk(get(src["path"], src.get("params")), src["pointer"])
                elif t == "ca":
                    cas = get(f"{F}/spec/certificate_authorities", {"include_secrets": "true"})["certificate_authorities"]
                    item = cas[src["kind"]]
                    item = item[src["index"]] if isinstance(item, list) else item
                    val = item[src["field"]]
                elif t == "mask":
                    val = given(name)
                    from_user = bool(val)
                    if not val:
                        # Masked values are bare `key: $NAME` in YAML, where an unquoted "*" starts an alias.
                        val = f'"{MASK}"'
                        notes.append(f"{name}: Fleet only returns this value masked; the dry run used Fleet's mask, which Fleet treats as unchanged")
                else:
                    val = given(name)
                    from_user = bool(val)
                    if not val:
                        val = PLACEHOLDER
                        notes.append(f"{name}: not set in .env or the environment; the dry run used a placeholder, so files using it weren't checked with the real value")
            except (KeyError, IndexError, TypeError, ValueError, OSError, subprocess.SubprocessError, FleetctlError) as ex:
                val = PLACEHOLDER
                notes.append(f"{name}: couldn't read it from Fleet ({ex}); the dry run used a placeholder")
            # Webhook URLs and IdP metadata are variables too, but a script or profile may use the same URL.
            secret = t in ("enroll_secret", "ca", "mask", "fleet_secret", "api") or \
                (t in ("config", "team_config") and (src.get("pointer") or [""])[0] in ("agent_options", "features"))
            if secret and isinstance(val, str) and val not in (PLACEHOLDER, f'"{MASK}"', MASK):
                self.secret_values[name] = val  # in memory only, for check_leaks
            # fleetctl substitutes values into YAML as-is (bare `key: $NAME`), so one with YAML syntax in it
            # reads back as something else. Multi-line metadata, JSON objects, and FLEET_SECRET_ values
            # (which fleetctl leaves for the server) are meant to go in raw.
            used_in = e.get("used_in") or []
            in_yaml = any(u.endswith((".yml", ".yaml")) for u in used_in) and t != "fleet_secret" \
                and not (src.get("indent") or src.get("object") or src.get("field") == "api_key_json")
            # Profiles and DDM assets that hold the variable get the value as-is, so quotes would end up in them.
            raw = [u for u in used_in if not u.endswith((".yml", ".yaml"))]
            if in_yaml and isinstance(val, str) and val != PLACEHOLDER and val != f'"{MASK}"' and not self._survives_yaml(val):
                if raw:
                    self.fail("reconcile", f"{name}: its value doesn't read back as the same text when fleetctl puts it in YAML as-is "
                                           "(it has YAML syntax in it, or reads as a number, boolean, or null), and quoting it would put "
                                           f"the quotes into {', '.join(raw)}. In Fleet, change it to a value that's plain text in YAML, "
                                           "then export again")
                elif not from_user:
                    val = json.dumps(val, ensure_ascii=False)
                    self.warn("reconcile", f"{name}: Fleet's value doesn't read back as the same text when fleetctl puts it in YAML as-is "
                                           "(it has YAML syntax in it, or reads as a number, boolean, or null). Set it in CI and .env "
                                           "wrapped in double quotes, with \\ and \" escaped; the dry run did that")
                elif not (len(val) > 1 and val[0] == val[-1] and val[0] in "'\""):
                    self.fail("reconcile", f"{name}: the value from .env or the environment doesn't read back as the same text when "
                                           "fleetctl puts it in YAML as-is (it has YAML syntax in it, or reads as a number, boolean, or null), "
                                           "so applying would set something else. Wrap it in double quotes, with \\ and \" escaped, in CI and .env")
            if src.get("indent") and isinstance(val, str) and "\n" in val:
                # Same trick as the commented-out lines in .github/fleet-gitops/gitops.sh: keep a multi-line value inside its YAML key.
                lines = val.split("\n")
                val = "\n".join([lines[0]] + [" " * src["indent"] + ln for ln in lines[1:]])
            env[name] = "" if val is None else str(val)
        return env, notes

    def check_leaks(self, work: Path | None = None, env_file: Path | None = None):
        """Fail when a repo file holds a value the export keeps in a variable (warn for the work folder).

        The export never writes these, so a hit means an edit or an export bug. Needs the values the dry run fetched.
        """
        needles = [(n, v.encode(e)) for n, v in self.secret_values.items() if len(v) >= MIN_SECRET_LEN for e in ("utf-8", "utf-16-le")]
        if not needles:
            return
        skip = {(self.root / ".env").resolve()} | ({Path(env_file).resolve()} if env_file else set())

        def held(p: Path) -> list:
            try:
                data = p.read_bytes()
            except OSError:
                return []
            return sorted({n for n, b in needles if b in data})

        for p in sorted(self.root.rglob("*")):
            rel = str(p.relative_to(self.root)).replace(os.sep, "/")
            if p.is_file() and not rel.startswith(".git/") and p.resolve() not in skip:
                for n in held(p):
                    self.fail("reconcile", f"{rel} holds the value of ${n} in plain text; don't commit it. The export writes it as "
                                           "a variable, so remove it from that file (or from Fleet, then export again)")
        if work:
            for p in [work / "plan.json"] + sorted((work / "blobs").rglob("*") if (work / "blobs").is_dir() else []):
                if p.is_file():
                    for n in held(p):
                        self.warn("reconcile", f"the work folder's {p.relative_to(work)} holds the value of ${n}; run `clean` when you're done")

    def _survives_yaml(self, val: str) -> bool:
        """Does `key: <val>` parse back to the string val? Assumed so without a YAML parser."""
        if not self.backend:
            return True
        try:
            return load_yaml("v: " + val, self.backend) == {"v": val}
        except Exception:  # noqa: BLE001 - either parser's error
            return False

    def dry_run(self, api: API, fleetctl, env_file: Path | None = None):
        if self.env_values is None:
            self.env_values, self.env_notes = self.dry_run_env(api, env_file)
        env, notes = self.env_values, self.env_notes
        files = ["default.yml"] + sorted("fleets/" + p.name for p in (self.root / "fleets").glob("*.yml"))
        try:
            r = fleetctl.gitops_dry_run(files, cwd=str(self.root), env=env)
        except (OSError, subprocess.SubprocessError, FleetctlError) as e:
            self.fail("reconcile", f"fleetctl gitops --dry-run couldn't run: {e}")
            return
        self.compared = True
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
        if not self.compared:
            self.fail("reconcile", "nothing was compared with Fleet: the reconcile check needs PyYAML or Ruby, and the dry run was skipped")
        levels = [r["level"] for k in ("references", "reconcile") for r in self.results[k]]
        self.results["passed"] = "FAIL" not in levels
        self.results["yaml_backend"] = self.backend
        return self.results
