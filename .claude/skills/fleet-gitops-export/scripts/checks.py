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
from planner import ANY_OF, TOP_LEVEL, VARIABLE

F = "/api/latest/fleet"
PLACEHOLDER = "fleet-gitops-export-dry-run-placeholder"
MASK = "********"
UNCHECKED = "(not compared)"

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
              "profiles": "profile", "assets": "DDM asset", "software": "software", "settings": "settings"}
LABEL_KEYS = ("labels_include_any", "labels_exclude_any", "labels_include_all")
# Settings whose omitted value isn't false.
OMITTED = {("controls", "enable_profile_validation"): True}
# Settings keyed by kind, where the repo's other keys are read too: a kind only the repo has is an addition.
KEYED_SECTIONS = ("org_settings.integrations", "org_settings.certificate_authorities")


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
        """(folder paths in it are relative to, item dict) under a section: entries of referenced item files,
        or the entries themselves when inline."""
        out = []
        for path, entry in self._items(top, section):
            if path and path.endswith((".yml", ".yaml")):
                out.extend((os.path.dirname(path), item) for item in self.docs.get(path) or [] if isinstance(item, dict))
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
                m = re.fullmatch(r"\$\{?(\w+)\}?", x) if isinstance(x, str) else None
                src = self.env_sources.get(m.group(1)) if m else None
                if src is None:
                    out[k] = self._strip_env(x)
                elif "kind" in src:
                    out[k] = {VARIABLE: f"{src['kind']}[{src['index']}].{src['field']}"}
            return out
        if isinstance(v, list):
            return [self._strip_env(x) for x in v]
        return v

    def _ref(self, base: str, ref) -> str | None:
        """The repo path a `{path: ...}` value points at."""
        if isinstance(ref, dict) and ref.get("path"):
            return os.path.normpath(os.path.join(base, ref["path"])).replace(os.sep, "/")
        return None

    def _script_sha(self, base: str, ref):
        path = self._ref(base, ref)
        return _sha(self._bytes(path, False)) if path and (self.root / path).is_file() else None

    def _query(self, base: str, ref):
        items = self.docs.get(self._ref(base, ref) or "") or []
        return items[0].get("query") if items and isinstance(items[0], dict) else None

    def _app_config(self, base: str, ref, platform: str):
        path = self._ref(base, ref)
        if not path or not (self.root / path).is_file():
            return None
        if platform in ("ios", "ipados"):
            return _sha(self._bytes(path, False))
        body = self._bytes(path, False)
        try:
            return json.loads(body)
        except ValueError:
            return body.decode(errors="replace")

    def _software_fields(self, scripts: dict, base: str, entry: dict) -> dict:
        """A software item's settings: its own, else the fleet-level entry's (as fleetctl merges them),
        with its scripts and query as reconcile compares them (scripts by hash)."""
        f = {k: scripts.get(k) or entry.get(k) for k in ("self_service", "categories", "display_name")}
        labels = {k: scripts[k] for k in LABEL_KEYS if scripts.get(k)} or {k: entry[k] for k in LABEL_KEYS if entry.get(k)}
        f.update({k: labels.get(k) for k in LABEL_KEYS})
        for key in ("install_script", "uninstall_script", "post_install_script"):
            f[key] = self._script_sha(base, scripts.get(key))
        f["pre_install_query"] = self._query(base, scripts.get("pre_install_query"))
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
            items = self.docs.get(self._ref(base, {"path": inst["package_path"]})) or []
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
                values[k] = OMITTED.get((section, k)) if v is None and (section, k) in OMITTED else self._strip_env(v)
            if section in KEYED_SECTIONS and isinstance(node, dict):
                values.update({k: self._strip_env(v) for k, v in node.items() if k not in fields})
            found["settings"].append([section, values])
        for path, _ in self._items(top, ["controls", "scripts"]):
            if path:
                found["scripts"].append([os.path.basename(path), {"sha256": _sha(self._bytes(path, False))}])
        for section in (["controls", "apple_settings", "configuration_profiles"], ["controls", "windows_settings", "configuration_profiles"],
                        ["controls", "android_settings", "configuration_profiles"]):
            for path, entry in self._items(top, section):
                if not path:
                    continue
                body = self._expanded_bytes(path)
                name = entry.get("name")
                if not name and path.endswith(".mobileconfig"):
                    name = _payload_display_name(self._bytes(path, True))
                name = name or os.path.splitext(os.path.basename(path))[0]
                f = {k: entry.get(k) for k in ("self_service", "hidden", "description", "labels_include_all", "labels_include_any", "labels_exclude_any")}
                f["sha256"] = UNCHECKED if body is None else _sha(body)
                if entry.get("activation"):
                    f["activation_sha256"] = _sha(self._bytes(os.path.normpath(os.path.join(base, entry["activation"])), False))
                found["profiles"].append([name, f])
        for path, _ in self._items(top, ["controls", "apple_settings", "assets"]):
            if path:
                body = self._expanded_bytes(path)
                found["assets"].append([os.path.splitext(os.path.basename(path))[0], {"sha256": UNCHECKED if body is None else _sha(body)}])
        sw = doc.get("software") or {}
        for entry in sw.get("packages") or []:
            if not isinstance(entry, dict):
                continue
            path = self._ref(base, entry)
            if path and path.endswith((".yml", ".yaml")):
                for item in self.docs.get(path) or []:
                    found["software"].append([f"package:{item.get('url') or item.get('hash_sha256')}",
                                              self._software_fields(item, os.path.dirname(path), entry)])
            elif path:
                found["software"].append([f"package:{_sha(self._bytes(path, False))}", self._software_fields(entry, base, entry)])
            else:
                found["software"].append([f"package:{entry.get('hash_sha256')}", self._software_fields(entry, base, entry)])
        for item in sw.get("fleet_maintained_apps") or []:
            if isinstance(item, dict):
                f = self._software_fields(item, base, item)
                f["version"] = item.get("version")
                found["software"].append([f"fma:{item.get('slug')}", f])
        for item in sw.get("app_store_apps") or []:
            if not isinstance(item, dict):
                continue
            ident, platform = f"app:{item.get('app_store_id')}:{item.get('platform', '')}", item.get("platform", "")
            if item.get("versions"):
                found["software"].append([ident, {"display_name": item.get("display_name")}])
            versions = [(f"{ident}:{v.get('name')}", v) for v in item.get("versions") or [] if isinstance(v, dict)] or [(ident, item)]
            for key, v in versions:
                f = {k: v.get(k) for k in ("self_service", "categories", "display_name") + LABEL_KEYS}
                f["configuration"] = self._app_config(base, v.get("configuration"), platform)
                found["software"].append([key, f])
        return found

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
        for top in planned + [t for t in inventory if t not in planned]:
            kinds = inventory.get(top) or {}
            if top not in self.docs:
                self.fail("reconcile", f"{top} is missing (applying would remove what Fleet has for it)")
                continue
            found = self._found(top, kinds.get("settings") or [])
            managed = [k for k in KIND_NAMES if not exceptions.get(k)]
            for kind in managed + [k for k in kinds if k not in managed]:
                want_list = kinds.get(kind) or []
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
                    extra = sorted(set(got[n]) - set(want[n])) if kind == "settings" else []
                    for field in extra:
                        if _norm(got[n][field]) is not None:
                            self.fail("reconcile", f"{top}: {what} {n!r}: {field} is in the repo but not in Fleet (applying would create it)")
                    for field, fleet_value in want[n].items():
                        if got[n].get(field) == UNCHECKED:
                            self.warn("reconcile", f"{top}: {what} {n!r}: {field} not compared; the file holds variables "
                                                   "whose values validate only fetches when it runs the dry run")
                            continue
                        if not _same(fleet_value, got[n].get(field)):
                            self.fail("reconcile", f"{top}: {what} {n!r}: {field} is {_short(got[n].get(field))} in the repo "
                                                   f"but {_short(fleet_value)} in Fleet (applying would change it)")

    # ------------------------------------------------------------ dry run

    def dry_run_env(self, api: API, env_file: Path | None = None) -> tuple[dict, list]:
        env = dict(os.environ)
        notes = []
        cache: dict = {}
        # Agents often run commands in a shell the user can't export into, so values only the
        # user has come from the repo's gitignored .env first.
        local = read_env_file(env_file) if env_file else {}

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
            except (KeyError, IndexError, TypeError, FleetctlError) as ex:
                val = PLACEHOLDER
                notes.append(f"{name}: couldn't read it from Fleet ({ex}); the dry run used a placeholder")
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
