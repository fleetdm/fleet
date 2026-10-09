#!/usr/bin/env python3
"""Static checks for a Fleet GitOps repository, without a Fleet server.

Usage:
  validate.py [ROOT] [--offline] [--ddf-dir DIR]
  validate.py --table NAME

ROOT is the directory containing default.yml (default: found from the current
directory). Exit status is 1 when anything FAILs.

Needs Python 3.8+ and nothing else. YAML is parsed with PyYAML when installed,
otherwise with Ruby's YAML (ships with macOS); without either, only the checks
that don't need parsed YAML run. Reference data (Fleet's osquery schema, the
Fleet-maintained app catalog, Google's Android Management API schema) is read
from the Fleet source tree when the repo is inside it, otherwise downloaded and
cached for a week under ~/.cache/fleet-gitops-skill. --offline skips downloads.
"""

import argparse
import difflib
import glob
import json
import os
import plistlib
import re
import subprocess
import sys
import time
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path

OSQUERY_SCHEMA_URL = "https://raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json"
FMA_CATALOG_URL = "https://raw.githubusercontent.com/fleetdm/fleet/main/ee/maintained-apps/outputs/apps.json"
ANDROID_DISCOVERY_URL = "https://androidmanagement.googleapis.com/$discovery/rest?version=v1"
CACHE_DIR = Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache")) / "fleet-gitops-skill"
CACHE_MAX_AGE = 7 * 24 * 3600

TOP_LEVEL_KEYS = {"name", "settings", "org_settings", "agent_options", "controls", "policies", "reports",
                  "software", "labels", "custom_host_vitals"}
DEPRECATED_TOP_LEVEL = {"team_settings": "settings", "queries": "reports"}
POLICY_PLATFORMS = {"darwin", "windows", "linux", "chrome"}
LABEL_PLATFORMS = {"darwin", "windows", "linux", "ubuntu", "centos"}
LABEL_PLATFORM_ALIASES = {"ubuntu": "linux", "centos": "linux"}
REPORT_LOGGING = {"snapshot", "differential", "differential_ignore_removals"}
LABEL_KEYS = ("labels_include_any", "labels_include_all", "labels_exclude_any", "labels_exclude_all")
AUTOMATION_KEYS = ("run_script", "install_software", "resend_configuration_profile", "calendar_events_enabled")
PATH_KEYS = {"path", "package_path", "apple_setup_assistant", "end_user_license_agreement", "activation"}
GLOB_CHARS = set("*?[{")
SCRIPT_EXTS = {".sh", ".ps1", ".py"}
BUILTIN_LABELS = {"All Hosts", "macOS", "Ubuntu Linux", "CentOS Linux", "MS Windows", "Red Hat Linux", "All Linux",
                  "chrome", "macOS 14+ (Sonoma+)", "iOS", "iPadOS", "Fedora Linux", "Android"}
RESERVED_PROFILE_NAMES = {"Fleetd configuration", "Fleet root certificate authority (CA)", "Disk encryption",
                          "Windows OS Updates", "Fleet macOS OS Updates", "Fleet iOS OS Updates",
                          "Fleet iPadOS OS Updates", "Fleetd enroll secret"}
RESERVED_PAYLOAD_IDS = {"com.fleetdm.fleet.mdm.filevault", "com.fleetdm.fleetd.config", "com.fleetdm.caroot"}
RESERVED_PAYLOAD_TYPES = {"com.apple.MCX.FileVault2": "disk encryption is managed by controls.enable_disk_encryption",
                          "com.apple.security.FDERecoveryKeyEscrow": "recovery key escrow is managed by Fleet"}
FORBIDDEN_DECL_TYPES = {"com.apple.configuration.watch.enrollment", "com.apple.configuration.account.google",
                        "com.apple.management.server-capabilities",
                        "com.apple.configuration.management.status-subscriptions", "com.apple.configuration.package"}
FLEET_MANAGED_DECL_TYPES = {"com.apple.configuration.softwareupdate.enforcement.specific": "controls.macos_updates / ios_updates / ipados_updates"}
WINDOWS_TOP_LEVEL = {"Replace", "Add", "Exec", "Atomic"}
WINDOWS_RESERVED_LOCURI = {"/Vendor/MSFT/BitLocker": "controls.enable_disk_encryption",
                           "/Vendor/MSFT/Policy/Config/Update": "controls.windows_updates",
                           "/Vendor/MSFT/RemoteWipe": "Fleet's wipe action"}
ANDROID_FORBIDDEN = {"statusReportingSettings", "applications", "appFunctions", "playStoreMode", "installAppsDisabled",
                     "uninstallAppsDisabled", "blockApplicationsEnabled", "appAutoUpdatePolicy",
                     "kioskCustomLauncherEnabled", "kioskCustomization", "persistentPreferredActivities",
                     "setupActions", "encryptionPolicy"}
ANDROID_PREMIUM = {"systemUpdate"}
SQL_ALIAS_STOPWORDS = {"where", "on", "join", "left", "right", "inner", "cross", "natural", "outer", "group", "order",
                       "limit", "union", "using", "having", "and", "or", "not", "as", "set", "when", "then", "else",
                       "end", "select", "from", "values", "intersect", "except", "with"}


# ---------------------------------------------------------------- reporting

class Ctx:
    def __init__(self, root, offline=False, ddf_dir=None):
        self.root = root
        self.offline = offline
        self.ddf_dir = ddf_dir
        self.findings = []  # (level, relpath, message)
        self.yaml_cache = {}
        self.yaml_backend = detect_yaml_backend()
        self._schema = None
        self._catalog = None
        self._android = None
        self._ddf_text = None
        self.label_refs = []        # (name, relfile, what, fleet)
        self.labels_declared = {}   # fleet or "" -> {name: relfile}
        self.labels_section_seen = False
        self.profile_identifiers = {}  # identifier -> [(fleet, relfile)]
        self.profile_uuids = {}        # uuid -> [relfile]
        self.referenced = set()        # resolved paths some config file points at

    def rel(self, p):
        try:
            return str(Path(p).resolve().relative_to(self.root.resolve()))
        except ValueError:
            return str(p)

    def add(self, level, path, msg):
        self.findings.append((level, self.rel(path) if path else "", msg))

    def fail(self, path, msg):
        self.add("FAIL", path, msg)

    def warn(self, path, msg):
        self.add("WARN", path, msg)

    def info(self, path, msg):
        self.add("INFO", path, msg)

    # -- reference data, loaded lazily --------------------------------------
    def schema(self):
        if self._schema is None:
            data = load_reference(self, "osquery_fleet_schema.json", ["schema/osquery_fleet_schema.json"], OSQUERY_SCHEMA_URL)
            self._schema = {}
            for t in data or []:
                self._schema[t["name"]] = {
                    "platforms": set(t.get("platforms") or []),
                    "columns": {c["name"] for c in t.get("columns", [])},
                    "evented": bool(t.get("evented")),
                }
            if not self._schema:
                self.warn("", "osquery schema unavailable (offline?): table and column checks skipped")
                self._schema = False
        return self._schema or None

    def catalog(self):
        if self._catalog is None:
            data = load_reference(self, "apps.json", ["ee/maintained-apps/outputs/apps.json"], FMA_CATALOG_URL)
            self._catalog = {a["slug"]: a for a in (data or {}).get("apps", [])} if data else False
            if not self._catalog:
                self.warn("", "Fleet-maintained app catalog unavailable (offline?): slug checks skipped")
        return self._catalog or None

    def android_schemas(self):
        if self._android is None:
            data = load_reference(self, "android_discovery.json", [], ANDROID_DISCOVERY_URL)
            self._android = (data or {}).get("schemas") or False
            if not self._android:
                self.warn("", "Android Management API schema unavailable (offline?): Android key and enum checks skipped")
        return self._android or None

    def ddf_text(self):
        if self._ddf_text is None:
            chunks = []
            if self.ddf_dir:
                for f in Path(self.ddf_dir).rglob("*.xml"):
                    try:
                        chunks.append(f.read_text(errors="replace"))
                    except OSError:
                        pass
            self._ddf_text = "\n".join(chunks)
        return self._ddf_text


def load_reference(ctx, cache_name, local_candidates, url):
    """A reference file: from the Fleet source tree if we're inside it, else a week-old cache, else download."""
    for anc in [ctx.root, *ctx.root.resolve().parents]:
        for cand in local_candidates:
            p = anc / cand
            if p.is_file():
                try:
                    return json.loads(p.read_text())
                except (OSError, ValueError):
                    pass
    cache = CACHE_DIR / cache_name
    if cache.is_file() and time.time() - cache.stat().st_mtime < CACHE_MAX_AGE:
        try:
            return json.loads(cache.read_text())
        except ValueError:
            pass
    if ctx.offline:
        return None
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            body = resp.read()
        data = json.loads(body)
        CACHE_DIR.mkdir(parents=True, exist_ok=True)
        cache.write_bytes(body)
        return data
    except Exception as e:  # noqa: BLE001 - any network or parse failure just disables the check
        ctx.info("", f"couldn't download {url}: {e}")
        if cache.is_file():
            try:
                return json.loads(cache.read_text())
            except ValueError:
                pass
        return None


# ---------------------------------------------------------------- YAML

def detect_yaml_backend():
    try:
        import yaml  # noqa: F401
        return "pyyaml"
    except ImportError:
        pass
    try:
        subprocess.run(["ruby", "-ryaml", "-e", "1"], check=True, capture_output=True, timeout=20)
        return "ruby"
    except (OSError, subprocess.SubprocessError):
        return None


def parse_yaml(text, backend):
    if backend == "pyyaml":
        import yaml
        return yaml.safe_load(text)
    if backend == "ruby":
        script = ("require 'yaml'; require 'json'; "
                  "d = YAML.respond_to?(:unsafe_load) ? YAML.unsafe_load(STDIN.read) : YAML.load(STDIN.read); "
                  "puts JSON.generate(d)")
        out = subprocess.run(["ruby", "-e", script], input=text, capture_output=True, text=True, timeout=60)
        if out.returncode != 0:
            raise ValueError(out.stderr.strip().splitlines()[-1] if out.stderr.strip() else "ruby YAML parse failed")
        return json.loads(out.stdout) if out.stdout.strip() else None
    raise RuntimeError("no YAML parser available")


def load_yaml_file(ctx, path):
    key = str(Path(path).resolve())
    if key in ctx.yaml_cache:
        return ctx.yaml_cache[key]
    data = None
    try:
        text = Path(path).read_text()
        data = parse_yaml(text, ctx.yaml_backend)
        if looks_like_unquoted_time(text):
            pass  # reported by the field checks, which see the parsed int
    except (OSError, ValueError, RuntimeError) as e:
        ctx.fail(path, f"YAML doesn't parse: {e}")
    except Exception as e:  # noqa: BLE001 - PyYAML raises its own hierarchy
        ctx.fail(path, f"YAML doesn't parse: {str(e).splitlines()[0]}")
    ctx.yaml_cache[key] = data
    return data


def looks_like_unquoted_time(text):
    return re.search(r"^\s*auto_update_window_(start|end):\s*\d{1,2}:\d{2}\s*$", text, re.M) is not None


# ---------------------------------------------------------------- helpers

def resolve_refs(ctx, src, entry, what="entry"):
    """Resolve an entry's path:/paths: against the file that contains it. Returns existing Paths."""
    base = Path(src).parent
    out = []
    if "path" in entry and "paths" in entry:
        ctx.fail(src, f"{what} has both 'path' and 'paths'; use one")
    p = entry.get("path")
    if p is not None:
        if not isinstance(p, str) or not p:
            ctx.fail(src, f"{what}: 'path' must be a non-empty string")
        elif p.startswith("$"):
            ctx.info(src, f"{what}: path {p} is an environment variable; not resolved")
        elif GLOB_CHARS & set(p):
            ctx.fail(src, f"{what}: 'path: {p}' contains glob characters; use 'paths:' for patterns")
        else:
            full = (base / p)
            if full.is_file():
                out.append(full.resolve())
                ctx.referenced.add(full.resolve())
            else:
                ctx.fail(src, f"{what}: path {p} doesn't exist (resolved relative to this file: {ctx.rel(full)})")
    g = entry.get("paths")
    if g is not None:
        if not isinstance(g, str) or not g:
            ctx.fail(src, f"{what}: 'paths' must be a non-empty glob string")
        else:
            matches = sorted(m for m in glob.glob(str(base / g)) if Path(m).is_file())
            if not matches:
                ctx.warn(src, f"{what}: paths glob {g} matches no files")
            out.extend(Path(m).resolve() for m in matches)
            ctx.referenced.update(Path(m).resolve() for m in matches)
    return out


ARTIFACT_DIRS = {"policies", "reports", "queries", "labels", "software", "scripts", "configuration-profiles",
                 "declaration-profiles", "enrollment-profiles", "managed-app-configurations", "icons"}
ARTIFACT_EXTS = {".yml", ".yaml", ".sh", ".ps1", ".py", ".mobileconfig", ".json", ".xml", ".png"}


def report_unreferenced(ctx):
    """Artifact files no config file points at do nothing; say so, since a stray copy is a common source of 'I changed it but nothing happened'."""
    config = {p.resolve() for p in config_files(ctx.root)}
    for f in sorted(ctx.root.rglob("*")):
        if not f.is_file() or f.suffix.lower() not in ARTIFACT_EXTS or f.resolve() in config or f.resolve() in ctx.referenced:
            continue
        parts = set(f.relative_to(ctx.root).parts[:-1])
        if any(part in (".git", "node_modules", ".github", ".claude", ".contour") for part in parts):
            continue
        if not parts & ARTIFACT_DIRS:
            continue
        ctx.info(f, "not referenced by any config file (directly or through a paths: glob); it does nothing until it is")


def expand_items(ctx, src, items, section):
    """A section list may mix inline items and path/paths references to files holding lists of items."""
    out = []
    if items is None:
        return out
    if not isinstance(items, list):
        ctx.fail(src, f"'{section}' must be a list")
        return out
    for it in items:
        if not isinstance(it, dict):
            ctx.fail(src, f"'{section}' entries must be mappings, got {type(it).__name__}")
            continue
        if "path" in it or "paths" in it:
            opts = {k: v for k, v in it.items() if k not in ("path", "paths")}
            for f in resolve_refs(ctx, src, it, section):
                data = load_yaml_file(ctx, f)
                if data is None:
                    continue
                entries = data if isinstance(data, list) else [data]
                for e in entries:
                    if not isinstance(e, dict):
                        ctx.fail(f, f"{section} file entries must be mappings")
                        continue
                    if "path" in e or "paths" in e:
                        ctx.fail(f, f"nested path references aren't supported in {section} files")
                        continue
                    out.append((e, f, opts))
        else:
            out.append((it, src, {}))
    return out


def walk_paths(ctx, src, node, what="file"):
    """Resolve every path-like key anywhere under node, relative to src. Catches controls, org_settings, package files."""
    if isinstance(node, dict):
        for k, v in node.items():
            if k in PATH_KEYS and isinstance(v, str):
                if v.startswith("$") or v.startswith("http://") or v.startswith("https://"):
                    continue
                resolve_refs(ctx, src, {"path": v}, f"{what} '{k}'")
            elif k == "paths" and isinstance(v, str):
                resolve_refs(ctx, src, {"paths": v}, f"{what} '{k}'")
            else:
                walk_paths(ctx, src, v, what)
    elif isinstance(node, list):
        for v in node:
            walk_paths(ctx, src, v, what)


def check_label_keys(ctx, src, item, what, fleet):
    inc_all = item.get("labels_include_all") or []
    inc_any = item.get("labels_include_any") or []
    exc_any = item.get("labels_exclude_any") or []
    exc_all = item.get("labels_exclude_all") or []
    for key in LABEL_KEYS:
        v = item.get(key)
        if v is not None and (not isinstance(v, list) or not all(isinstance(x, str) for x in v)):
            ctx.fail(src, f"{what}: '{key}' must be a list of label names")
    if inc_all and inc_any:
        ctx.fail(src, f"{what}: only one of labels_include_all or labels_include_any is allowed")
    overlap = set(map(str, inc_all + inc_any)) & set(map(str, exc_any + exc_all))
    if overlap:
        ctx.fail(src, f"{what}: label(s) {sorted(overlap)} appear in both an include and an exclude list")
    for name in inc_all + inc_any + exc_any + exc_all:
        if isinstance(name, str):
            ctx.label_refs.append((name, ctx.rel(src), what, fleet))


def check_platform(ctx, src, item, what, allowed, required=False):
    p = item.get("platform")
    if p is None or p == "":
        if required:
            ctx.warn(src, f"{what}: no 'platform'; the query will run on every platform")
        return set()
    if not isinstance(p, str):
        ctx.fail(src, f"{what}: 'platform' must be a string such as darwin, windows, linux")
        return set()
    parts = {x.strip() for x in p.split(",") if x.strip()}
    bad = parts - allowed
    if bad:
        ctx.fail(src, f"{what}: unknown platform(s) {sorted(bad)}; allowed: {sorted(allowed)}")
    return {LABEL_PLATFORM_ALIASES.get(x, x) for x in parts - bad}


# ---------------------------------------------------------------- osquery SQL

def strip_sql(sql):
    sql = re.sub(r"--[^\n]*", " ", sql)
    sql = re.sub(r"/\*.*?\*/", " ", sql, flags=re.S)
    sql = re.sub(r"'(?:[^']|'')*'", "''", sql)
    sql = re.sub(r'"(?:[^"]|"")*"', '""', sql)
    return sql


def extract_tables(sql):
    """Return {table_name: {aliases}} for every FROM/JOIN source that isn't a CTE or subquery."""
    s = strip_sql(sql)
    ctes = {m.group(1).lower() for m in re.finditer(r"(?i)\b([a-z_][a-z0-9_]*)\s+AS\s*\(\s*SELECT", s)}
    tables = {}
    pattern = re.compile(r"(?i)\b(FROM|JOIN)\s+([a-z_][a-z0-9_]*)(?:\s+(?:AS\s+)?([a-z_][a-z0-9_]*))?"
                         r"((?:\s*,\s*[a-z_][a-z0-9_]*(?:\s+(?:AS\s+)?[a-z_][a-z0-9_]*)?)*)")
    for m in pattern.finditer(s):
        sources = [(m.group(2), m.group(3))]
        for extra in re.finditer(r",\s*([a-z_][a-z0-9_]*)(?:\s+(?:AS\s+)?([a-z_][a-z0-9_]*))?", m.group(4) or "", re.I):
            sources.append((extra.group(1), extra.group(2)))
        for name, alias in sources:
            name = name.lower()
            if name in ctes:
                continue
            if alias and alias.lower() in SQL_ALIAS_STOPWORDS:
                alias = None
            tables.setdefault(name, set())
            if alias:
                tables[name].add(alias.lower())
    return tables


def check_query(ctx, src, what, sql, platforms, is_policy=False):
    schema = ctx.schema()
    if not isinstance(sql, str) or not sql.strip():
        ctx.fail(src, f"{what}: 'query' is empty")
        return
    if schema is None:
        return
    tables = extract_tables(sql)
    if not tables:
        if not re.search(r"(?i)\bselect\b", sql):
            ctx.fail(src, f"{what}: query doesn't look like SQL")
        return
    targets = platforms or {"darwin", "windows", "linux"}
    alias_to_table = {}
    for name, aliases in tables.items():
        t = schema.get(name)
        if t is None:
            hint = difflib.get_close_matches(name, schema.keys(), n=2)
            ctx.fail(src, f"{what}: table '{name}' isn't in Fleet's osquery schema" + (f" (did you mean {', '.join(hint)}?)" if hint else ""))
            continue
        alias_to_table[name] = name
        for a in aliases:
            alias_to_table[a] = name
        missing = sorted(p for p in targets if p not in t["platforms"])
        if missing:
            level = ctx.fail if platforms else ctx.warn
            level(src, f"{what}: table '{name}' isn't available on {', '.join(missing)}"
                  + ("" if platforms else " (set 'platform' to where this query should run)")
                  + f"; it exists on {', '.join(sorted(t['platforms']))}")
        if t["evented"] and is_policy:
            ctx.warn(src, f"{what}: '{name}' is an evented table; policies see only events buffered since the last run")
    for m in re.finditer(r"\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\b", strip_sql(sql), re.I):
        qual, col = m.group(1).lower(), m.group(2).lower()
        t = alias_to_table.get(qual)
        if t and col not in schema[t]["columns"] and col != "rowid":
            hint = difflib.get_close_matches(col, schema[t]["columns"], n=2)
            ctx.fail(src, f"{what}: column '{col}' isn't in table '{t}'" + (f" (did you mean {', '.join(hint)}?)" if hint else ""))


# ---------------------------------------------------------------- profiles

def neutralize_placeholders(text):
    return re.sub(r"\$\{?[A-Z][A-Z0-9_]*\}?|%[A-Za-z]+%", "AAAA", text)


def check_mobileconfig(ctx, f, entry_name, fleet):
    raw = f.read_bytes()
    text = raw.decode("utf-8", "replace")
    try:
        pl = plistlib.loads(raw)
    except Exception:  # noqa: BLE001
        try:
            pl = plistlib.loads(neutralize_placeholders(text).encode())
            ctx.info(f, "contains $VARIABLE or %placeholder% values; parsed with them neutralized")
        except Exception as e:  # noqa: BLE001
            ctx.fail(f, f"not a valid XML plist: {str(e).splitlines()[0]}")
            return None
    if not isinstance(pl, dict):
        ctx.fail(f, "top level of a .mobileconfig must be a dict")
        return None
    if pl.get("PayloadType") != "Configuration":
        ctx.fail(f, f"top-level PayloadType must be 'Configuration' (got {pl.get('PayloadType')!r})")
    for k in ("PayloadIdentifier", "PayloadUUID", "PayloadDisplayName"):
        if not pl.get(k):
            ctx.fail(f, f"top-level {k} is missing")
    if "PayloadVersion" not in pl:
        ctx.warn(f, "top-level PayloadVersion is missing (use 1)")
    ident = pl.get("PayloadIdentifier")
    name = entry_name or pl.get("PayloadDisplayName")
    if ident in RESERVED_PAYLOAD_IDS:
        ctx.fail(f, f"PayloadIdentifier {ident} is reserved by Fleet")
    if name in RESERVED_PROFILE_NAMES:
        ctx.fail(f, f"profile name {name!r} is reserved by Fleet")
    if "$FLEET_SECRET_" in text:
        ctx.info(f, "references $FLEET_SECRET_; fleetctl gitops --dry-run skips this profile")
    content = pl.get("PayloadContent")
    if not isinstance(content, list) or not content:
        ctx.warn(f, "PayloadContent is empty; this profile configures nothing")
        content = []
    for i, p in enumerate(content):
        if not isinstance(p, dict):
            ctx.fail(f, f"PayloadContent[{i}] isn't a dict")
            continue
        for k in ("PayloadType", "PayloadIdentifier", "PayloadUUID"):
            if k not in p:
                ctx.fail(f, f"PayloadContent[{i}] ({p.get('PayloadType', '?')}) is missing {k}")
        if "PayloadVersion" not in p:
            ctx.warn(f, f"PayloadContent[{i}] ({p.get('PayloadType', '?')}) is missing PayloadVersion; Apple's spec requires it (use 1)")
        pt = p.get("PayloadType")
        if pt in RESERVED_PAYLOAD_TYPES:
            ctx.fail(f, f"PayloadContent[{i}] uses {pt}: {RESERVED_PAYLOAD_TYPES[pt]}")
        if p.get("PayloadIdentifier") in RESERVED_PAYLOAD_IDS:
            ctx.fail(f, f"PayloadContent[{i}] PayloadIdentifier {p.get('PayloadIdentifier')} is reserved by Fleet")
    if ident:
        ctx.profile_identifiers.setdefault(ident, []).append((fleet, ctx.rel(f)))
    for u in [pl.get("PayloadUUID")] + [p.get("PayloadUUID") for p in content if isinstance(p, dict)]:
        if u:
            ctx.profile_uuids.setdefault(u, []).append(ctx.rel(f))
    return name


def check_declaration(ctx, f, entry_name, fleet):
    try:
        d = json.loads(f.read_text())
    except (OSError, ValueError) as e:
        ctx.fail(f, f"not valid JSON: {e}")
        return None
    if not isinstance(d, dict):
        ctx.fail(f, "a declaration must be a JSON object")
        return None
    for k in ("Type", "Identifier", "Payload"):
        if k not in d:
            ctx.fail(f, f"declaration is missing '{k}'")
    t = d.get("Type", "")
    if t in FORBIDDEN_DECL_TYPES:
        ctx.fail(f, f"declaration type {t} is rejected by Fleet")
    elif t in FLEET_MANAGED_DECL_TYPES:
        ctx.fail(f, f"declaration type {t} is managed by Fleet; use {FLEET_MANAGED_DECL_TYPES[t]}")
    elif t and not t.startswith(("com.apple.configuration.", "com.apple.management.", "com.apple.asset.")):
        ctx.fail(f, f"declaration type {t} isn't a configuration, management, or asset declaration (activations are built by Fleet)")
    ident = d.get("Identifier", "")
    if isinstance(ident, str):
        if len(ident.encode()) > 64:
            ctx.fail(f, "Identifier is longer than 64 bytes; devices reject it at delivery")
        if ident and "." not in ident:
            ctx.warn(f, f"Identifier {ident!r} isn't reverse-DNS; follow the repo's identifier prefix")
        if ident:
            ctx.profile_identifiers.setdefault(ident, []).append((fleet, ctx.rel(f)))
    if "ServerToken" in d:
        ctx.warn(f, "ServerToken is set by Fleet; remove it")
    if not isinstance(d.get("Payload"), dict) or not d.get("Payload"):
        ctx.warn(f, "Payload is empty; this declaration configures nothing")
    return entry_name or f.stem


def local(tag):
    return tag.split("}", 1)[-1]


def check_windows_profile(ctx, f, entry_name, fleet):
    text = f.read_text(errors="replace")
    try:
        root = ET.fromstring("<fleet-wrapper>" + text + "</fleet-wrapper>")
    except ET.ParseError as e:
        ctx.fail(f, f"not well-formed XML: {e}")
        return None
    tops = [local(c.tag) for c in root]
    if not tops:
        ctx.fail(f, "no SyncML commands found; expected top-level <Replace>, <Add>, <Exec>, or <Atomic>")
    bad = sorted({t for t in tops if t not in WINDOWS_TOP_LEVEL})
    if bad:
        ctx.fail(f, f"unsupported top-level element(s) {bad}; Fleet accepts <Replace>, <Add>, <Exec>, <Atomic> (no root element)")
    if "$FLEET_SECRET_" in text:
        ctx.info(f, "references $FLEET_SECRET_; fleetctl gitops --dry-run skips this profile")
    uris = []
    for item in root.iter():
        if local(item.tag) != "Item":
            continue
        loc = None
        for el in item.iter():
            if local(el.tag) == "LocURI":
                loc = (el.text or "").strip()
        has_format = any(local(el.tag) == "Format" for el in item.iter())
        has_data = any(local(el.tag) == "Data" for el in item.iter())
        if loc is None:
            ctx.fail(f, "an <Item> has no <Target><LocURI>")
            continue
        uris.append(loc)
        if not loc.startswith("./"):
            ctx.fail(f, f"LocURI {loc} must start with './' (e.g. ./Device/Vendor/MSFT/...)")
        if "../" in loc:
            ctx.fail(f, f"LocURI {loc} contains '../'")
        for reserved, owner in WINDOWS_RESERVED_LOCURI.items():
            if reserved.lower() in loc.lower():
                ctx.fail(f, f"LocURI {loc} targets a node Fleet manages; use {owner}")
        if not has_format:
            ctx.warn(f, f"<Item> for {loc} has no <Meta><Format>; set it to the node's DDF DFFormat (int, chr, bool, b64)")
        if not has_data:
            ctx.warn(f, f"<Item> for {loc} has no <Data>")
    ddf = ctx.ddf_text()
    if ddf:
        for loc in uris:
            node = loc.rsplit("/", 1)[-1]
            if f"<NodeName>{node}</NodeName>" not in ddf:
                ctx.fail(f, f"LocURI {loc}: node '{node}' not found in the DDF files under {ctx.ddf_dir}")
    return entry_name or f.stem


def check_android_profile(ctx, f, entry_name, fleet):
    try:
        d = json.loads(f.read_text())
    except (OSError, ValueError) as e:
        ctx.fail(f, f"not valid JSON: {e}")
        return None
    if not isinstance(d, dict):
        ctx.fail(f, "an Android profile must be a JSON object (a subset of the Policy resource)")
        return None
    if not d:
        ctx.fail(f, "profile is empty")
    for k in d:
        if k in ANDROID_FORBIDDEN:
            ctx.fail(f, f"'{k}' is rejected by Fleet (managed elsewhere or unsupported)")
        elif k in ANDROID_PREMIUM:
            ctx.warn(f, f"'{k}' needs Fleet Premium")
    schemas = ctx.android_schemas()
    if schemas and "Policy" in schemas:
        check_android_values(ctx, f, d, "Policy", schemas, top=True, path="")
    return entry_name or f.stem


def check_android_values(ctx, f, obj, schema_name, schemas, top, path):
    props = schemas.get(schema_name, {}).get("properties", {})
    for k, v in obj.items():
        here = f"{path}.{k}" if path else k
        p = props.get(k)
        if p is None:
            if k in ANDROID_FORBIDDEN or k in ANDROID_PREMIUM:
                continue
            if top:
                ctx.fail(f, f"'{here}' isn't a field of the Android Policy resource; Fleet rejects unknown top-level keys")
            else:
                ctx.fail(f, f"'{here}' isn't a field of {schema_name}; Google drops unknown nested keys silently")
            continue
        enum = p.get("enum") or (p.get("items") or {}).get("enum")
        if enum:
            values = v if isinstance(v, list) else [v]
            for val in values:
                if val not in enum:
                    hint = difflib.get_close_matches(str(val), enum, n=1, cutoff=0.5)
                    ctx.fail(f, f"'{here}': {val!r} isn't an allowed value" + (f" (did you mean {hint[0]}?)" if hint else ""))
        ptype = p.get("type")
        if ptype == "boolean" and not isinstance(v, bool):
            ctx.fail(f, f"'{here}' must be true or false")
        elif ptype == "string" and not isinstance(v, str):
            ctx.fail(f, f"'{here}' must be a string")
        elif ptype == "integer" and (not isinstance(v, int) or isinstance(v, bool)):
            ctx.fail(f, f"'{here}' must be an integer")
        elif ptype == "array" and not isinstance(v, list):
            ctx.fail(f, f"'{here}' must be a list")
        ref = p.get("$ref") or (p.get("items") or {}).get("$ref")
        if ref:
            children = v if isinstance(v, list) else [v]
            for child in children:
                if isinstance(child, dict):
                    check_android_values(ctx, f, child, ref, schemas, top=False, path=here)


# ---------------------------------------------------------------- sections

class FleetState:
    def __init__(self, name, role, src):
        self.name = name
        self.role = role  # global | fleet | unassigned
        self.src = src
        self.scripts = set()         # resolved paths
        self.package_files = set()   # resolved paths
        self.package_hashes = set()
        self.fma_slugs = set()
        self.app_store_ids = set()
        self.profile_names = {}      # name -> relfile
        self.policy_names = {}
        self.report_names = {}


def check_controls(ctx, fs, controls):
    src = fs.src
    if controls is None:
        return
    if not isinstance(controls, dict):
        ctx.fail(src, "'controls' must be a mapping")
        return
    walk_paths(ctx, src, {k: v for k, v in controls.items() if k not in ("scripts", "apple_settings", "macos_settings", "windows_settings", "android_settings")}, "controls")
    scripts = controls.get("scripts")
    if isinstance(scripts, list):
        for entry in scripts:
            if not isinstance(entry, dict):
                ctx.fail(src, "controls.scripts entries must be mappings with path or paths")
                continue
            for f in resolve_refs(ctx, src, entry, "controls.scripts"):
                if f.suffix not in SCRIPT_EXTS:
                    ctx.fail(src, f"controls.scripts: {ctx.rel(f)} must be a .sh, .ps1, or .py file")
                elif f.suffix == ".sh" and not f.read_text(errors="replace").startswith("#!"):
                    ctx.warn(f, "shell script has no shebang; add one (for example #!/bin/sh)")
                fs.scripts.add(f)
    elif scripts is not None:
        ctx.fail(src, "controls.scripts must be a list")
    for key, kind in (("apple_settings", "apple"), ("macos_settings", "apple"), ("windows_settings", "windows"), ("android_settings", "android")):
        section = controls.get(key)
        if not isinstance(section, dict):
            continue
        walk_paths(ctx, src, {k: v for k, v in section.items() if k != "configuration_profiles"}, f"controls.{key}")
        profiles = section.get("configuration_profiles")
        if profiles is None:
            continue
        if not isinstance(profiles, list):
            ctx.fail(src, f"controls.{key}.configuration_profiles must be a list")
            continue
        for entry in profiles:
            if not isinstance(entry, dict):
                ctx.fail(src, f"controls.{key}.configuration_profiles entries must be mappings")
                continue
            what = f"controls.{key}.configuration_profiles"
            check_label_keys(ctx, src, entry, what, fs.name)
            if "name" in entry and "paths" in entry:
                ctx.fail(src, f"{what}: 'name' can't be combined with a 'paths' glob")
            if "activation" in entry:
                resolve_refs(ctx, src, {"path": entry["activation"]}, f"{what} activation")
            for f in resolve_refs(ctx, src, entry, what):
                name = None
                ext = f.suffix.lower()
                if kind == "apple" and ext == ".mobileconfig":
                    name = check_mobileconfig(ctx, f, entry.get("name"), fs.name)
                elif kind == "apple" and ext == ".json":
                    name = check_declaration(ctx, f, entry.get("name"), fs.name)
                elif kind == "windows" and ext == ".xml":
                    name = check_windows_profile(ctx, f, entry.get("name"), fs.name)
                elif kind == "android" and ext == ".json":
                    name = check_android_profile(ctx, f, entry.get("name"), fs.name)
                else:
                    ctx.fail(src, f"{what}: {ctx.rel(f)} has the wrong extension for {kind} ({ext})")
                if name:
                    if name in fs.profile_names:
                        ctx.fail(src, f"profile name {name!r} is used twice in this fleet ({fs.profile_names[name]} and {ctx.rel(f)}); names must be unique per fleet across platforms")
                    fs.profile_names[name] = ctx.rel(f)


def check_software(ctx, fs, software):
    src = fs.src
    if software is None:
        return
    if fs.role == "global":
        ctx.fail(src, "'software' can't be set in default.yml; software is configured per fleet")
        return
    if not isinstance(software, dict):
        ctx.fail(src, "'software' must be a mapping")
        return
    unknown = set(software) - {"packages", "app_store_apps", "fleet_maintained_apps"}
    if unknown:
        ctx.fail(src, f"unknown software key(s) {sorted(unknown)}; allowed: packages, app_store_apps, fleet_maintained_apps")

    packages = software.get("packages")
    if isinstance(packages, list):
        for entry in packages:
            if not isinstance(entry, dict):
                ctx.fail(src, "software.packages entries must be mappings")
                continue
            what = "software.packages"
            check_label_keys(ctx, src, entry, what, fs.name)
            check_bool(ctx, src, entry, "self_service", what)
            check_bool(ctx, src, entry, "setup_experience", what)
            if "icon" in entry:
                walk_paths(ctx, src, {"icon": entry["icon"]}, what)
            for f in resolve_refs(ctx, src, entry, what):
                if f.suffix in SCRIPT_EXTS:
                    fs.package_files.add(f)
                    if "install_script" in entry:
                        ctx.fail(src, f"{what}: {f.name} is itself the install script; 'install_script' doesn't apply to script-only packages")
                    walk_paths(ctx, src, {k: v for k, v in entry.items() if k in ("uninstall_script", "post_install_script", "pre_install_query")}, what)
                    if "display_name" not in entry:
                        ctx.warn(src, f"{what}: script-only package {f.name} has no display_name; the file name will be shown")
                    continue
                if f.suffix not in (".yml", ".yaml"):
                    ctx.fail(src, f"{what}: {ctx.rel(f)} must be a package .yml or a .sh/.ps1/.py script")
                    continue
                fs.package_files.add(f)
                data = load_yaml_file(ctx, f)
                if data is None:
                    continue
                pkgs = data if isinstance(data, list) else [data]
                for pkg in pkgs:
                    if not isinstance(pkg, dict):
                        ctx.fail(f, "package entries must be mappings")
                        continue
                    check_package(ctx, f, pkg, fs)
    elif packages is not None:
        ctx.fail(src, "software.packages must be a list")

    for item, isrc, opts in expand_items(ctx, src, software.get("app_store_apps"), "software.app_store_apps"):
        merged = {**opts, **item}
        what = f"software.app_store_apps {merged.get('app_store_id', '?')}"
        aid = merged.get("app_store_id")
        if not isinstance(aid, str) or not aid:
            ctx.fail(isrc, f"{what}: app_store_id must be a quoted string (for example \"361285480\" or \"com.android.chrome\")")
        plat = merged.get("platform")
        if plat is not None and plat not in {"darwin", "ios", "ipados", "android"}:
            ctx.fail(isrc, f"{what}: platform must be darwin, ios, ipados, or android")
        key = (str(aid), plat)
        if key in fs.app_store_ids:
            ctx.fail(isrc, f"{what}: listed twice for platform {plat or 'all'}")
        fs.app_store_ids.add(key)
        check_label_keys(ctx, isrc, merged, what, fs.name)
        check_bool(ctx, isrc, merged, "self_service", what)
        check_bool(ctx, isrc, merged, "setup_experience", what)
        check_categories(ctx, isrc, merged, what)
        if merged.get("auto_update_enabled"):
            if plat not in (None, "ios", "ipados"):
                ctx.fail(isrc, f"{what}: auto_update_enabled is only supported for ios and ipados apps")
            for k in ("auto_update_window_start", "auto_update_window_end"):
                v = merged.get(k)
                if not isinstance(v, str) or not re.fullmatch(r"\d{2}:\d{2}", v):
                    ctx.fail(isrc, f"{what}: {k} must be a quoted \"HH:MM\" string when auto_update_enabled is true (got {v!r})")
        walk_paths(ctx, isrc, {k: v for k, v in item.items() if k in ("icon", "configuration")}, what)

    seen_slugs = {}
    for item, isrc, opts in expand_items(ctx, src, software.get("fleet_maintained_apps"), "software.fleet_maintained_apps"):
        merged = {**opts, **item}
        slug = merged.get("slug")
        what = f"software.fleet_maintained_apps {slug!r}"
        if not isinstance(slug, str) or "/" not in slug:
            ctx.fail(isrc, f"{what}: slug must look like <app>/darwin or <app>/windows")
            continue
        if slug in seen_slugs:
            ctx.fail(isrc, f"{what}: listed twice in this fleet")
        seen_slugs[slug] = isrc
        fs.fma_slugs.add(slug)
        catalog = ctx.catalog()
        if catalog is not None and slug not in catalog:
            hint = difflib.get_close_matches(slug, catalog.keys(), n=3, cutoff=0.6)
            ctx.fail(isrc, f"{what}: not in the Fleet-maintained app catalog" + (f" (closest: {', '.join(hint)})" if hint else ""))
        v = merged.get("version")
        if v is not None and not isinstance(v, str):
            ctx.fail(isrc, f"{what}: version must be a quoted string (got {v!r}; YAML read it as {type(v).__name__})")
        elif isinstance(v, str) and not re.fullmatch(r"\^?\d+(\.\d+)*[A-Za-z0-9.+-]*", v):
            ctx.warn(isrc, f"{what}: version {v!r} doesn't look like a version or ^major constraint")
        check_label_keys(ctx, isrc, merged, what, fs.name)
        check_bool(ctx, isrc, merged, "self_service", what)
        check_bool(ctx, isrc, merged, "setup_experience", what)
        check_categories(ctx, isrc, merged, what)
        walk_paths(ctx, isrc, {k: v for k, v in item.items() if k in ("install_script", "uninstall_script", "post_install_script", "pre_install_query", "icon")}, what)


def check_package(ctx, f, pkg, fs):
    url = pkg.get("url")
    h = pkg.get("hash_sha256")
    if not url and not h:
        ctx.fail(f, "package needs 'url' or 'hash_sha256'")
    if h is not None:
        if not isinstance(h, str) or not re.fullmatch(r"[0-9a-f]{64}", h):
            ctx.fail(f, "hash_sha256 must be 64 lowercase hex characters")
        else:
            fs.package_hashes.add(h)
    if isinstance(url, str) and not url.startswith("$"):
        if not re.match(r"https?://", url):
            ctx.fail(f, f"url {url!r} isn't an http(s) URL")
        lower = url.split("?", 1)[0].lower()
        if (lower.endswith(".exe") or lower.endswith(".tar.gz")) and not (pkg.get("install_script") and pkg.get("uninstall_script")):
            ctx.fail(f, f"{Path(lower).name}: .exe and .tar.gz packages require both install_script and uninstall_script")
    if pkg.get("always_download") and h:
        ctx.fail(f, "always_download can't be combined with hash_sha256")
    for k in LABEL_KEYS:
        if k in pkg:
            ctx.fail(f, f"'{k}' belongs on the fleet file's packages entry, not in the package file")
    for k in ("self_service", "setup_experience", "categories"):
        if k in pkg:
            ctx.fail(f, f"'{k}' belongs on the fleet file's packages entry, not in the package file")
    walk_paths(ctx, f, pkg, "package")


def check_bool(ctx, src, item, key, what):
    if key in item and not isinstance(item[key], bool):
        ctx.fail(src, f"{what}: '{key}' must be true or false")


def check_categories(ctx, src, item, what):
    c = item.get("categories")
    if c is not None and (not isinstance(c, list) or not all(isinstance(x, str) for x in c)):
        ctx.fail(src, f"{what}: 'categories' must be a list of names")


def check_labels(ctx, fs, labels):
    src = fs.src
    if labels is None:
        return
    ctx.labels_section_seen = True
    if fs.role == "unassigned":
        ctx.fail(src, "'labels' can't be set in unassigned.yml")
        return
    scope = "" if fs.role == "global" else fs.name
    declared = ctx.labels_declared.setdefault(scope, {})
    for item, isrc, opts in expand_items(ctx, src, labels, "labels"):
        name = item.get("name")
        what = f"label {name!r}"
        if not isinstance(name, str) or not name:
            ctx.fail(isrc, "label needs a 'name'")
            continue
        if name in BUILTIN_LABELS:
            ctx.fail(isrc, f"{what}: this name is a built-in label")
        for other_scope, names in ctx.labels_declared.items():
            if name in names:
                ctx.fail(isrc, f"{what}: already declared in {names[name]}; label names are unique across the instance")
        declared[name] = ctx.rel(isrc)
        mtype = item.get("label_membership_type", "dynamic")
        if mtype not in ("dynamic", "manual", "host_vitals"):
            ctx.fail(isrc, f"{what}: label_membership_type must be dynamic, manual, or host_vitals")
        present = [k for k in ("query", "hosts", "criteria") if k in item]
        if len(present) > 1:
            ctx.fail(isrc, f"{what}: only one of query, hosts, criteria is allowed (got {present})")
        if mtype == "dynamic" and "query" not in item:
            ctx.fail(isrc, f"{what}: a dynamic label needs a 'query'")
        if mtype == "manual" and "query" in item:
            ctx.fail(isrc, f"{what}: a manual label can't have a 'query'")
        if mtype == "host_vitals" and not isinstance(item.get("criteria"), dict):
            ctx.fail(isrc, f"{what}: a host_vitals label needs 'criteria' with vital and value")
        if "hosts" in item and isinstance(item["hosts"], list):
            for h in item["hosts"]:
                if not isinstance(h, str):
                    ctx.fail(isrc, f"{what}: host identifiers must be quoted strings (got {h!r})")
        desc = item.get("description")
        if isinstance(desc, str) and len(desc) > 255:
            ctx.warn(isrc, f"{what}: description is {len(desc)} characters; fleetctl rejects more than 255")
        if "platform" in item and mtype != "dynamic":
            ctx.fail(isrc, f"{what}: 'platform' is only supported for dynamic labels")
        platforms = check_platform(ctx, isrc, item, what, LABEL_PLATFORMS)
        if "query" in item:
            check_query(ctx, isrc, what, item["query"], platforms)


def check_policies(ctx, fs, policies):
    src = fs.src
    if policies is None:
        return
    for item, isrc, opts in expand_items(ctx, src, policies, "policies"):
        if any(k in opts for k in LABEL_KEYS):
            ctx.warn(src, "labels on a 'policies' path entry don't apply; put labels_include_* on each policy in the file")
        name = item.get("name")
        what = f"policy {name!r}"
        if not isinstance(name, str) or not name:
            ctx.fail(isrc, "policy needs a 'name'")
            continue
        if name in fs.policy_names:
            ctx.fail(isrc, f"{what}: name already used in this fleet ({fs.policy_names[name]})")
        fs.policy_names[name] = ctx.rel(isrc)
        for k in ("description", "resolution"):
            if not item.get(k):
                ctx.warn(isrc, f"{what}: no '{k}'")
        check_bool(ctx, isrc, item, "critical", what)
        check_label_keys(ctx, isrc, item, what, fs.name)
        is_patch = item.get("type") == "patch"
        if "type" in item and not is_patch:
            ctx.fail(isrc, f"{what}: 'type' must be 'patch' or omitted")
        if is_patch:
            slug = item.get("fleet_maintained_app_slug")
            if not isinstance(slug, str):
                ctx.fail(isrc, f"{what}: a patch policy needs 'fleet_maintained_app_slug'")
            elif fs.role == "fleet" and slug not in fs.fma_slugs:
                ctx.fail(isrc, f"{what}: {slug} isn't in this fleet's software.fleet_maintained_apps")
            if item.get("query"):
                ctx.warn(isrc, f"{what}: patch policies generate their own query; 'query' is ignored")
            if (item.get("patch_when_closed") or item.get("notify_before_patching")) and item.get("install_software") is not True:
                ctx.fail(isrc, f"{what}: patch_when_closed / notify_before_patching need 'install_software: true', otherwise nothing is ever installed")
            if item.get("notify_before_patching") and slug and not str(slug).endswith("/darwin"):
                ctx.fail(isrc, f"{what}: notify_before_patching is macOS-only")
        else:
            platforms = check_platform(ctx, isrc, item, what, POLICY_PLATFORMS, required=True)
            if "query" not in item:
                ctx.fail(isrc, f"{what}: needs a 'query'")
            else:
                check_query(ctx, isrc, what, item["query"], platforms, is_policy=True)
        automations = [k for k in AUTOMATION_KEYS if item.get(k)]
        if fs.role == "global" and automations:
            ctx.fail(isrc, f"{what}: {automations} only work on policies in a fleet file, not in default.yml")
        rs = item.get("run_script")
        if rs is not None:
            if not isinstance(rs, dict) or not isinstance(rs.get("path"), str):
                ctx.fail(isrc, f"{what}: run_script must be a mapping with 'path'")
            else:
                for f in resolve_refs(ctx, isrc, rs, f"{what} run_script"):
                    if fs.role != "global" and f not in fs.scripts:
                        ctx.fail(isrc, f"{what}: run_script {ctx.rel(f)} must also be listed under this fleet's controls.scripts")
        inst = item.get("install_software")
        if inst is not None and not is_patch:
            if not isinstance(inst, dict):
                ctx.fail(isrc, f"{what}: install_software must be a mapping with one of package_path, fleet_maintained_app_slug, app_store_id, hash_sha256")
            else:
                keys = [k for k in ("package_path", "fleet_maintained_app_slug", "app_store_id", "hash_sha256") if k in inst]
                if len(keys) != 1:
                    ctx.fail(isrc, f"{what}: install_software needs exactly one of package_path, fleet_maintained_app_slug, app_store_id, hash_sha256 (got {keys})")
                if "package_path" in inst:
                    for f in resolve_refs(ctx, isrc, {"path": inst["package_path"]}, f"{what} install_software"):
                        if fs.role != "global" and f not in fs.package_files:
                            ctx.fail(isrc, f"{what}: install_software package {ctx.rel(f)} must also be listed under this fleet's software.packages")
                if "fleet_maintained_app_slug" in inst and fs.role != "global" and inst["fleet_maintained_app_slug"] not in fs.fma_slugs:
                    ctx.fail(isrc, f"{what}: install_software slug {inst['fleet_maintained_app_slug']} isn't in this fleet's software.fleet_maintained_apps")
                if "app_store_id" in inst and fs.role != "global" and str(inst["app_store_id"]) not in {a for a, _ in fs.app_store_ids}:
                    ctx.fail(isrc, f"{what}: install_software app_store_id {inst['app_store_id']} isn't in this fleet's software.app_store_apps")
                if "hash_sha256" in inst and fs.role != "global" and inst["hash_sha256"] not in fs.package_hashes:
                    ctx.warn(isrc, f"{what}: install_software hash {inst['hash_sha256']} doesn't match a package file in this fleet")
        rcp = item.get("resend_configuration_profile")
        if rcp is not None and fs.role != "global" and rcp not in fs.profile_names:
            ctx.fail(isrc, f"{what}: resend_configuration_profile {rcp!r} isn't a profile name in this fleet's controls (known: {sorted(fs.profile_names)[:8]})")


def check_reports(ctx, fs, reports):
    src = fs.src
    if reports is None:
        return
    if fs.role == "unassigned":
        ctx.fail(src, "'reports' can't be set in unassigned.yml")
        return
    for item, isrc, opts in expand_items(ctx, src, reports, "reports"):
        merged = {**opts, **item}
        name = item.get("name")
        what = f"report {name!r}"
        if not isinstance(name, str) or not name:
            ctx.fail(isrc, "report needs a 'name'")
            continue
        if name in fs.report_names:
            ctx.fail(isrc, f"{what}: name already used in this fleet ({fs.report_names[name]})")
        fs.report_names[name] = ctx.rel(isrc)
        platforms = check_platform(ctx, isrc, item, what, POLICY_PLATFORMS, required=True)
        if "query" not in item:
            ctx.fail(isrc, f"{what}: needs a 'query'")
        else:
            check_query(ctx, isrc, what, item["query"], platforms)
        interval = item.get("interval")
        if interval is None or interval == 0:
            ctx.warn(isrc, f"{what}: no 'interval'; the report never runs on a schedule")
        elif not isinstance(interval, int) or isinstance(interval, bool) or interval < 0:
            ctx.fail(isrc, f"{what}: interval must be a number of seconds")
        logging = item.get("logging")
        if logging is not None and logging not in REPORT_LOGGING:
            ctx.fail(isrc, f"{what}: logging must be one of {sorted(REPORT_LOGGING)}")
        for k in ("observer_can_run", "automations_enabled", "discard_data"):
            check_bool(ctx, isrc, item, k, what)
        if any(k in merged for k in ("labels_include_all", "labels_exclude_any")):
            ctx.fail(isrc, f"{what}: reports only support labels_include_any")
        check_label_keys(ctx, isrc, merged, what, fs.name)


def check_settings(ctx, fs, data):
    src = fs.src
    if fs.role == "global":
        org = data.get("org_settings")
        if not isinstance(org, dict):
            ctx.fail(src, "default.yml needs an 'org_settings' mapping")
        else:
            walk_paths(ctx, src, org, "org_settings")
            if "software" in data:
                pass  # reported by check_software
    else:
        if "org_settings" in data:
            ctx.fail(src, "'org_settings' only belongs in default.yml; fleet files use 'settings'")
    if "agent_options" in data:
        if fs.role == "unassigned":
            ctx.fail(src, "'agent_options' can't be set in unassigned.yml")
        walk_paths(ctx, src, {"agent_options": data["agent_options"]}, "agent_options")
    chv = data.get("custom_host_vitals")
    if chv is not None:
        if fs.role != "global":
            ctx.fail(src, "'custom_host_vitals' can only be set in default.yml")
        elif not isinstance(chv, list) or not all(isinstance(x, dict) and isinstance(x.get("name"), str) for x in chv):
            ctx.fail(src, "'custom_host_vitals' must be a list of {name: ...}")


# ---------------------------------------------------------------- driver

def find_root(start):
    start = Path(start).resolve()
    if (start / "default.yml").is_file():
        return start
    for anc in start.parents:
        if (anc / "default.yml").is_file():
            return anc
    found = []
    for depth in range(1, 5):
        for p in start.glob("/".join(["*"] * depth) + "/default.yml"):
            if any(part in (".git", "node_modules", "vendor") for part in p.parts):
                continue
            found.append(p.parent)
    if len(found) == 1:
        return found[0]
    if len(found) > 1:
        with_fleets = [f for f in found if (f / "fleets").is_dir() or (f / "teams").is_dir()]
        if len(with_fleets) == 1:
            return with_fleets[0]
        sys.exit("Several default.yml found; pass the GitOps root explicitly:\n  " + "\n  ".join(str(f) for f in found))
    sys.exit(f"No default.yml found under {start}; pass the GitOps root (the directory containing default.yml).")


def config_files(root):
    files = []
    if (root / "default.yml").is_file():
        files.append(root / "default.yml")
    for sub in ("fleets", "teams"):
        files.extend(sorted((root / sub).glob("*.yml")))
        files.extend(sorted((root / sub).glob("*.yaml")))
    for extra in ("no-team.yml", "unassigned.yml"):
        if (root / extra).is_file():
            files.append(root / extra)
    return files


def run_checks(ctx):
    files = config_files(ctx.root)
    if not ctx.yaml_backend:
        ctx.warn("", "no YAML parser (install PyYAML: pip install pyyaml, or Ruby); YAML-level checks skipped")
        for f in files:
            regex_only_checks(ctx, f)
        return
    fleet_names = {}
    states = []
    for src in files:
        data = load_yaml_file(ctx, src)
        if data is None:
            if src.exists() and not src.read_text().strip():
                ctx.fail(src, "file is empty")
            continue
        if not isinstance(data, dict):
            ctx.fail(src, "top level must be a mapping")
            continue
        for k in list(data):
            if k in DEPRECATED_TOP_LEVEL:
                ctx.warn(src, f"'{k}' is deprecated; use '{DEPRECATED_TOP_LEVEL[k]}'")
                data[DEPRECATED_TOP_LEVEL[k]] = data.pop(k)
        unknown = set(data) - TOP_LEVEL_KEYS
        if unknown:
            ctx.fail(src, f"unknown top-level key(s) {sorted(unknown)}; allowed: {sorted(TOP_LEVEL_KEYS)}")
        base = src.name
        if "org_settings" in data:
            role, name = "global", "(global)"
            if "name" in data or "settings" in data:
                ctx.fail(src, "'org_settings' can't be combined with 'name' or 'settings'")
            if base != "default.yml":
                ctx.warn(src, "org_settings found outside default.yml; CI usually only applies default.yml")
        elif "name" in data:
            name = data["name"]
            if not isinstance(name, str) or not name.strip():
                ctx.fail(src, "'name' must be a non-empty string")
                name = base
            role = "unassigned" if name.strip().lower() in ("unassigned", "no team") else "fleet"
            if role == "unassigned" and base not in ("unassigned.yml", "no-team.yml"):
                ctx.fail(src, "a fleet named 'Unassigned' must live in fleets/unassigned.yml")
            if role == "fleet" and base in ("unassigned.yml", "no-team.yml"):
                ctx.fail(src, f"{base} must have name: Unassigned")
            if base == "no-team.yml":
                ctx.warn(src, "no-team.yml is deprecated; rename to unassigned.yml with name: Unassigned")
            if role == "fleet":
                if name in fleet_names:
                    ctx.fail(src, f"fleet name {name!r} is also used by {fleet_names[name]}")
                fleet_names[name] = ctx.rel(src)
        else:
            ctx.fail(src, "file needs either 'org_settings' (default.yml) or 'name' (a fleet file)")
            continue
        fs = FleetState(name, role, src)
        states.append(fs)
        check_settings(ctx, fs, data)
        check_labels(ctx, fs, data.get("labels"))
        check_controls(ctx, fs, data.get("controls"))
        check_software(ctx, fs, data.get("software"))
        check_policies(ctx, fs, data.get("policies"))
        check_reports(ctx, fs, data.get("reports"))
    resolve_label_refs(ctx)
    report_unreferenced(ctx)
    for ident, uses in ctx.profile_identifiers.items():
        per_fleet = {}
        for fleet, rf in uses:
            per_fleet.setdefault(fleet, []).append(rf)
        for fleet, rfs in per_fleet.items():
            if len(set(rfs)) > 1:
                ctx.fail(rfs[0], f"identifier {ident} is used by {len(set(rfs))} profiles in fleet {fleet!r} ({', '.join(sorted(set(rfs)))}); devices treat them as the same profile")
    for u, rfs in ctx.profile_uuids.items():
        if len(set(rfs)) > 1:
            ctx.warn(sorted(set(rfs))[0], f"PayloadUUID {u} is reused across {', '.join(sorted(set(rfs)))}; generate a fresh UUID per payload")


def resolve_label_refs(ctx):
    global_labels = ctx.labels_declared.get("", {})
    for name, rf, what, fleet in ctx.label_refs:
        if name in BUILTIN_LABELS:
            ctx.fail(rf, f"{what}: '{name}' is a built-in label and can't be referenced; use 'platform' instead")
            continue
        if name in global_labels or name in ctx.labels_declared.get(fleet, {}):
            continue
        if not ctx.labels_section_seen:
            ctx.warn(rf, f"{what}: label '{name}' isn't declared in this repo; if labels are managed in the Fleet UI, confirm it exists there")
        else:
            known = [n for names in ctx.labels_declared.values() for n in names]
            hint = difflib.get_close_matches(name, known, n=2, cutoff=0.6)
            ctx.fail(rf, f"{what}: label '{name}' isn't declared in any labels: section" + (f" (did you mean {', '.join(repr(h) for h in hint)}?)" if hint else ""))


def regex_only_checks(ctx, src):
    text = src.read_text(errors="replace")
    base = src.parent
    for m in re.finditer(r"^\s*-?\s*(path|package_path):\s*([^\s#]+)", text, re.M):
        p = m.group(2).strip("'\"")
        if p.startswith("$"):
            continue
        if not (base / p).exists():
            ctx.fail(src, f"'{m.group(1)}: {p}' doesn't exist (relative to this file)")
    for m in re.finditer(r"^\s*(version|app_store_id|auto_update_window_start|auto_update_window_end):\s*(\d[\d.:]*)\s*$", text, re.M):
        ctx.fail(src, f"'{m.group(1)}: {m.group(2)}' must be quoted so YAML keeps it a string")


def print_table(name):
    ctx = Ctx(Path.cwd())
    schema = ctx.schema()
    if not schema:
        sys.exit("osquery schema unavailable")
    t = schema.get(name)
    if not t:
        hint = difflib.get_close_matches(name, schema.keys(), n=5)
        sys.exit(f"no table named {name!r}" + (f"; close matches: {', '.join(hint)}" if hint else ""))
    data = load_reference(ctx, "osquery_fleet_schema.json", ["schema/osquery_fleet_schema.json"], OSQUERY_SCHEMA_URL)
    entry = next(x for x in data if x["name"] == name)
    print(f"{name}  platforms: {', '.join(sorted(t['platforms']))}  evented: {t['evented']}")
    print(f"  {entry.get('description', '').strip()}")
    print(f"  https://fleetdm.com/tables/{name}")
    for c in entry.get("columns", []):
        flags = []
        if c.get("required"):
            flags.append("required")
        if c.get("platforms"):
            flags.append("on " + ",".join(c["platforms"]))
        print(f"  - {c['name']} ({c.get('type', '?')}){' [' + ', '.join(flags) + ']' if flags else ''}: {c.get('description', '').strip()}")


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("root", nargs="?", default=".", help="GitOps root (directory containing default.yml)")
    ap.add_argument("--offline", action="store_true", help="never download reference data")
    ap.add_argument("--ddf-dir", help="directory of Microsoft DDF XML files to check Windows LocURI node names against")
    ap.add_argument("--table", help="print an osquery table's platforms and columns, then exit")
    args = ap.parse_args()
    if args.table:
        print_table(args.table)
        return 0
    root = find_root(args.root)
    ctx = Ctx(root, offline=args.offline, ddf_dir=args.ddf_dir)
    run_checks(ctx)
    order = {"FAIL": 0, "WARN": 1, "INFO": 2}
    counts = {"FAIL": 0, "WARN": 0, "INFO": 0}
    for level, rel, msg in sorted(ctx.findings, key=lambda x: (order[x[0]], x[1], x[2])):
        counts[level] += 1
        print(f"{level:4} {rel + ': ' if rel else ''}{msg}")
    files = config_files(root)
    print(f"\nChecked {len(files)} config file(s) under {root} (YAML parser: {ctx.yaml_backend or 'none'}): "
          f"{counts['FAIL']} FAIL, {counts['WARN']} WARN, {counts['INFO']} INFO")
    return 1 if counts["FAIL"] else 0


if __name__ == "__main__":
    sys.exit(main())
