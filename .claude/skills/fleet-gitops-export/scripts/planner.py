"""Turn a snapshot of Fleet into an export plan.

The plan names every file to write (with a destination derived from the
scaffold), what goes into default.yml and each fleet file, every environment
variable the repo needs, and every item that needs a human look or can't be
represented. It holds no secret values. File references stay symbolic
({"$file": id}) until placement, so moving a file during review only means
editing its "path" in plan.json.
"""

from __future__ import annotations

import difflib
import hashlib
import json
import os
import plistlib
import re
import unicodedata
from pathlib import Path
from urllib.parse import unquote

from fleet_defaults import is_default_script
from layout import Layout
from yamlio import env_escape, fleet_secret_names

PLATFORM_DIR = {"darwin": "macos", "macos": "macos", "windows": "windows", "linux": "linux",
                "ios": "ios", "ipados": "ipados", "android": "android"}

MASK = "********"

# Deprecated names the API still returns next to the new ones (pkg/spec/gitops_deprecations.go).
# fleetctl gitops rejects a file that has both, so only the new name is kept.
ALIASES = {
    "server_settings": {"live_query_disabled": "live_reporting_disabled", "query_reports_disabled": "discard_reports_data",
                        "query_report_cap": "report_cap"},
    "org_info": {"org_logo_url": "org_logo_url_dark_mode", "org_logo_url_light_background": "org_logo_url_light_mode"},
    "apple_business": {"macos_team": "macos_fleet", "ios_team": "ios_fleet", "ipados_team": "ipados_fleet", "byod_team": "byod_fleet"},
    "volume_purchasing_program": {"teams": "fleets"},
    "settings": {"custom_settings": "configuration_profiles"},
}

# Paths sit outside the \b group: a word boundary never comes before "/" after a space or quote.
LINUX_HINTS = re.compile(r"\b(?:apt-get|apt|dpkg|yum|dnf|rpm|zypper|pacman|snap|flatpak|systemctl|journalctl|update-alternatives|lsb_release|"
                         r"gsettings|dconf|nmcli|loginctl|udevadm|udev|modprobe|blockdev|lsblk|ubuntu|debian|fedora|rhel|centos)\b"
                         r"|/etc/os-release|/etc/systemd/|/etc/udev/|/etc/apt/|/proc/|/sys/", re.I)
MAC_HINTS = re.compile(r"\b(?:launchctl|defaults|osascript|profiles|diskutil|softwareupdate|pkgutil|installer|scutil|fdesetup|sw_vers|mdmclient|"
                       r"dscl|plutil|csrutil|spctl|systemsetup|networksetup|sysadminctl|hdiutil|xattr|codesign|macos|darwin)\b"
                       r"|/Library/|/Applications/|/System/", re.I)

# An older copy of an app's catalog script scores 0.83 to 0.93 against the current one (measured on
# dogfood after the uninstall template changed); a script an admin rewrote scores far lower.
FMA_DRIFT_RATIO = 0.75

UNSAFE_CHARS = set('/\\:*?"<>|[{')


def slugify(name: str, fallback: str = "item") -> str:
    out = []
    for c in unicodedata.normalize("NFKD", name or ""):
        if c.isascii() and c.isalnum():
            out.append(c.lower())
        elif c.isascii() and (c.isspace() or c in "-_./:+&,()"):
            out.append("-")
    s = re.sub(r"-+", "-", "".join(out)).strip("-")
    return s or fallback


def exact_name_ok(name: str) -> bool:
    """Can this be a file name on macOS, Linux, and Windows, and in a GitOps `path:`?"""
    if not name or len(name.encode()) > 200 or name != name.strip() or name.endswith("."):
        return False
    if any(c in UNSAFE_CHARS or ord(c) < 32 for c in name):
        return False
    stem = name.split(".")[0].upper()
    return stem not in {"CON", "PRN", "AUX", "NUL"} and not re.fullmatch(r"(COM|LPT)\d", stem)


def drop_aliases(section: str, d):
    if isinstance(d, list):
        return [drop_aliases(section, x) for x in d]
    if not isinstance(d, dict):
        return d
    rules = ALIASES.get(section, {})
    out = {}
    for k, v in d.items():
        if k in rules:
            new = rules[k]
            if new in d:
                continue
            k = new
        out[k] = v
    return out


def names(labels) -> list[str]:
    return [lbl.get("name") or lbl.get("label_name") for lbl in (labels or [])]


def scope_labels(obj: dict, allowed=("labels_include_all", "labels_include_any", "labels_exclude_any", "labels_exclude_all")) -> dict:
    return {k: names(obj[k]) for k in allowed if obj.get(k)}


def software_labels(obj: dict) -> dict:
    # Software takes one label list; Fleet stores at most one, checked in this order.
    for k in ("labels_include_any", "labels_exclude_any", "labels_include_all"):
        if obj.get(k):
            return {k: names(obj[k])}
    return {}


def text_of(b: bytes) -> str | None:
    try:
        return b.decode("utf-8")
    except UnicodeDecodeError:
        return None


class Planner:
    def __init__(self, snap: dict, layout: Layout, work: Path, keep_fma_scripts=()):
        self.snap = snap
        self.layout = layout
        self.work = work
        self.blobs = work / "blobs"
        self.blobs.mkdir(parents=True, exist_ok=True)
        self.flags = snap["flags"]
        self.exceptions = (snap.get("gitops") or {}).get("exceptions") or {}
        # Fleet-maintained app slugs (or "all") whose stored scripts are kept even when they look like older catalog copies.
        self.keep_fma_scripts = set(keep_fma_scripts)
        self.files: dict = {}
        self._dedupe: dict = {}
        self._paths: dict = {}
        self.env: dict = {}
        self.notes: list = []
        self.fleets: list = []
        self.remove: list = []
        self.counts: dict = {}
        self._older_fma_scripts: dict = {}
        self._clashes = None

    # ------------------------------------------------------------ bookkeeping

    def note(self, level, scope, item, message):
        self.notes.append({"level": level, "scope": scope, "item": item, "message": message})

    def count(self, scope, kind, n=1):
        self.counts.setdefault(scope, {}).setdefault(kind, 0)
        self.counts[scope][kind] += n

    def add_env(self, name: str, where: str | None, description: str, source: dict) -> dict:
        base, i = name, 2
        while name in self.env and self.env[name]["source"] != source:
            name, i = f"{base}_{i}", i + 1
        entry = self.env.setdefault(name, {"name": name, "description": description, "source": source, "used_in": []})
        if where and where not in entry["used_in"]:
            entry["used_in"].append(where)
        return {"$env": name}

    def _enroll_secret_name(self, team_id, index: int) -> tuple[str, str]:
        if team_id is None:
            name = "FLEET_GLOBAL_ENROLL_SECRET"
            description = "Enroll secret for hosts in Unassigned" if self.flags["premium"] else "Enroll secret"
        else:
            scope = next((s["name"] for s in self.snap["scopes"] if s["id"] == team_id), str(team_id))
            name, description = f"FLEET_{_envslug(scope)}_ENROLL_SECRET", f"Enroll secret for {scope}"
        return (name + f"_{index + 1}" if index else name), description

    def _enroll_secret_env(self, team_id, index: int, where: str | None) -> dict:
        name, description = self._enroll_secret_name(team_id, index)
        return self.add_env(name, where, description, {"type": "enroll_secret", "team": team_id, "index": index})

    def _content_with_env_refs(self, item: dict) -> tuple[bytes, bool, list]:
        """A profile or asset body with the enroll secrets collect found in it written as $VARIABLEs.

        Returns (content, escape, variable names). When there's a variable, the content comes back
        already escaped, because escaping at placement would turn the inserted $NAME into a literal.
        """
        refs, text = item.get("env_refs") or [], text_of(item["content"])
        if not refs or text is None:
            return item["content"], text is not None, []
        out, pos, used = [], 0, []
        for r in sorted(refs, key=lambda r: r["start"]):
            src = r["source"]
            name = self._enroll_secret_env(src["team"], src["index"], None)["$env"]
            out.append(env_escape(text[pos:r["start"]]))
            follows = text[r["end"]:r["end"] + 1]
            out.append("${" + name + "}" if re.match(r"[A-Za-z0-9_]", follows) else "$" + name)
            pos = r["end"]
            if name not in used:
                used.append(name)
        out.append(env_escape(text[pos:]))
        return "".join(out).encode("utf-8"), False, used

    def _record_env_refs(self, fid: str, names: list, scope: str):
        if not names:
            return
        path = self.files[fid]["path"]
        self.files[fid]["env_refs"] = names
        for name in names:
            if path not in self.env[name]["used_in"]:
                self.env[name]["used_in"].append(path)
        self.note("info", scope, path, "Fleet's copy holds an enroll secret, so it's written as " + ", ".join("$" + n for n in names))

    def add_file(self, kind: str, platform: str | None, name: str, *, content: bytes | None = None, data=None,
                 exact: bool = False, ext: str = "", directory: str | None = None, fleet: str = "",
                 escape: bool = False, review: str | None = None, label: str = "") -> str:
        if directory is None:
            directory, origin = self.layout.dir_for(kind, platform)
        else:
            origin = "folder" if directory in self.layout.data["dirs"] else "new"
        filename = name if exact else slugify(name) + ext
        payload = content if content is not None else json.dumps(data, sort_keys=True, ensure_ascii=False).encode()
        digest = hashlib.sha256(payload).hexdigest()
        key = (directory, filename, digest, escape)
        if key in self._dedupe:
            fid = self._dedupe[key]
            if fleet and fleet not in self.files[fid]["fleets"]:
                self.files[fid]["fleets"].append(fleet)
            return fid

        path = f"{directory}/{filename}"
        if path.lower() in self._paths:
            fleet_slug = slugify(fleet, "fleet") if fleet else "2"
            if exact:
                sub, n = fleet_slug, 2
                while f"{directory}/{sub}/{filename}".lower() in self._paths:
                    sub, n = f"{fleet_slug}-{n}", n + 1
                path = f"{directory}/{sub}/{filename}"
            else:
                stem, n = f"{slugify(name)}-{fleet_slug}", 2
                path = f"{directory}/{stem}{ext}"
                while path.lower() in self._paths:
                    path, n = f"{directory}/{stem}-{n}{ext}", n + 1
        fid = f"f{len(self.files) + 1}"
        entry = {"path": path, "kind": kind, "platform": platform, "label": label or name, "fleets": [fleet] if fleet else [],
                 "origin": origin, "escape": escape, "review": review}
        if content is not None:
            (self.blobs / digest).write_bytes(content)
            entry["blob"] = digest
        else:
            entry["data"] = data
        self.files[fid] = entry
        self._dedupe[key] = fid
        self._paths[path.lower()] = fid
        if review:
            self.note("review", fleet or "Global", path, review)
        for secret in sorted(fleet_secret_names(text_of(content or b"") or "")):
            self.add_env(secret, path, "Custom variable stored in Fleet; Fleet can't return its value, so copy it from where you keep it", {"type": "fleet_secret"})
        return fid

    # ------------------------------------------------------------ top level

    def build(self) -> dict:
        snap, cfg = self.snap, self.snap["config"]
        premium = self.flags["premium"]
        scopes = {s["key"]: s for s in snap["scopes"]}
        self._map_fleets(scopes)

        default = self._fleet_entry("global")
        default["sections"]["org_settings"] = self._org_settings(cfg)
        if cfg.get("agent_options"):
            default["sections"]["agent_options"] = cfg["agent_options"]
            self.count("Global", "agent options")
        vitals = [{"name": v["name"]} for v in snap.get("custom_host_vitals") or []]
        if vitals:
            default["sections"]["custom_host_vitals"] = vitals
            self.count("Global", "custom host vitals", len(vitals))

        controls = self._global_controls(cfg["mdm"])
        controls_scope = scopes["unassigned"] if premium else scopes["global"]
        controls.update(self._fleet_controls(cfg["mdm"], controls_scope, default))
        default["sections"]["controls"] = controls

        g = scopes["global"]
        self._labels(g, default)
        self._policies(g, default, cfg.get("webhook_settings") or {})
        self._reports(g, default)

        if premium:
            un = scopes["unassigned"]
            unassigned = self._fleet_entry("unassigned")
            unassigned["sections"]["name"] = "Unassigned"
            team = un.get("team") or {}
            wh = drop_aliases("", team.get("webhook_settings") or {})
            unassigned_wh = {}
            for k in ("failing_policies_webhook", "host_activities_webhook"):
                if wh.get(k) and any(v for kk, v in wh[k].items() if kk != "policy_ids"):
                    unassigned_wh[k] = self._webhook(k, wh[k], "Unassigned", "fleets/unassigned.yml")
            if unassigned_wh:
                unassigned["sections"]["settings"] = {"webhook_settings": unassigned_wh}
            self._software(un, unassigned)
            self._policies(un, unassigned, team.get("webhook_settings") or {})

            for key, sc in scopes.items():
                if key in ("global", "unassigned"):
                    continue
                self._team(sc)

        self._examples()
        for scope, items in self._older_fma_scripts.items():
            self.note("review", scope, "Fleet-maintained app scripts",
                      f"left out {len(items)} stored scripts that differ only slightly from each app's current Fleet script: {', '.join(items)}. "
                      "They look like older copies of Fleet's own script that the app's next update would replace, and applying the repo "
                      "replaces them with the current one. Fleet doesn't report whether an admin edited a script, so ask the user; to keep "
                      "edited ones, re-run export with --keep-fma-scripts SLUG[,SLUG...]")
        return {
            "version": 1,
            "server": {"version": snap["version"], "premium": premium, "user": snap["me"]},
            "flags": self.flags,
            "exceptions": self.exceptions,
            "files": self.files,
            "fleets": self.fleets,
            "env": sorted(self.env.values(), key=lambda e: e["name"]),
            "notes": self.notes + [{"level": "warn", **w} for w in snap.get("warnings", [])],
            "remove": sorted(set(self.remove)),
            "counts": self.counts,
            "inventory": self._inventory(scopes),
        }

    # ------------------------------------------------------------ fleets

    def _map_fleets(self, scopes):
        templates = self.layout.fleet_templates
        by_name = {t["name"]: f for f, t in templates.items()}
        by_globs = sorted(templates, key=lambda f: len([g for g in self.layout.globs if g["file"] == f]))
        mobile_tpl = by_globs[0] if by_globs else None
        computer_tpl = by_globs[-1] if by_globs else None
        used, taken = set(), set()
        self._fleet_files = {}
        for key, sc in scopes.items():
            if key in ("global", "unassigned"):
                continue
            name = sc["name"]
            slug = slugify(name, "fleet")
            if name in by_name:
                fpath, action, tpl = by_name[name], "fill", by_name[name]
            elif f"fleets/{slug}.yml" in templates and f"fleets/{slug}.yml" not in used:
                fpath = tpl = f"fleets/{slug}.yml"
                action = "fill"
            else:
                fpath, n = f"fleets/{slug}.yml", 2
                while fpath in taken or fpath in templates or fpath == "fleets/unassigned.yml":
                    fpath, n = f"fleets/{slug}-{n}.yml", n + 1
                tpl = mobile_tpl if self._is_mobile(sc) else computer_tpl
                action = "clone"
            used.add(tpl)
            taken.add(fpath)
            self._fleet_files[key] = {"file": fpath, "template": tpl, "action": action}
        for f in templates:
            if f not in used:
                self.remove.append(f)
        if self.flags["premium"]:
            self._fleet_files["unassigned"] = {"file": "fleets/unassigned.yml", "template": None, "action": "create"}
        self._fleet_files["global"] = {"file": "default.yml", "template": "default.yml", "action": "fill"}

    def _is_mobile(self, sc) -> bool:
        computer, mobile = 0, 0
        computer += len(sc.get("scripts") or []) + len(sc.get("policies") or []) + len(sc.get("reports") or [])
        for p in sc.get("profiles") or []:
            if p.get("platform") == "windows":
                computer += 1
            elif p.get("platform") == "android":
                mobile += 1
        for t in (sc.get("software") or {}).get("titles", []):
            plat = ((t.get("software_package") or {}).get("platform") or (t.get("app_store_app") or {}).get("platform") or "")
            if plat in ("ios", "ipados", "android"):
                mobile += 1
            elif plat:
                computer += 1
        if computer or mobile:
            return mobile > 0 and computer == 0
        return bool(re.search(r"mobile|phone|ipad|iphone|android|byod", sc["name"], re.I))

    def _fleet_entry(self, key):
        info = self._fleet_files[key]
        entry = {"key": key, "file": info["file"], "template": info["template"], "action": info["action"],
                 "sections": {}, "entries": {}}
        self.fleets.append(entry)
        return entry

    def _scope_name(self, fleet) -> str:
        return {"global": "Global", "unassigned": "Unassigned"}.get(fleet["key"], fleet["sections"].get("name") or fleet["key"])

    def _team(self, sc):
        team = sc.get("team") or {}
        fleet = self._fleet_entry(sc["key"])
        fleet["name"] = sc["name"]
        fleet["sections"]["name"] = sc["name"]
        scope = sc["name"]
        settings = {}
        if team.get("features"):
            settings["features"] = team["features"]
        if team.get("host_expiry_settings"):
            settings["host_expiry_settings"] = team["host_expiry_settings"]
        wh = team.get("webhook_settings") or {}
        whs = {}
        for k, v in wh.items():
            if isinstance(v, dict):
                whs[k] = self._webhook(k, v, scope, fleet["file"])
        if whs:
            settings["webhook_settings"] = whs
        integ = team.get("integrations") or {}
        ti = {}
        if "conditional_access_enabled" in integ:
            ti["conditional_access_enabled"] = bool(integ["conditional_access_enabled"])
        gc = integ.get("google_calendar")
        if isinstance(gc, dict):
            gc = dict(gc)
            if gc.get("webhook_url"):
                gc["webhook_url"] = self.add_env(f"FLEET_{_envslug(scope)}_CALENDAR_WEBHOOK_URL", fleet["file"],
                                                 f"Google Calendar webhook URL for {scope}",
                                                 {"type": "team_config", "team": sc["id"], "pointer": ["integrations", "google_calendar", "webhook_url"]})
            ti["google_calendar"] = gc
        if integ.get("jira") or integ.get("zendesk"):
            self.note("gap", scope, "integrations", "Jira and Zendesk settings on a fleet can't be set in GitOps; they stay as they are in Fleet")
        if ti:
            settings["integrations"] = ti
        if not self.exceptions.get("secrets"):
            settings["secrets"] = [{"secret": self._enroll_secret_env(sc["id"], i, fleet["file"])}
                                   for i in range(sc.get("enroll_secret_count", 0))]
        fleet["sections"]["settings"] = settings
        if team.get("agent_options"):
            fleet["sections"]["agent_options"] = team["agent_options"]
            self.count(scope, "agent options")
        fleet["sections"]["controls"] = self._fleet_controls(team.get("mdm") or {}, sc, fleet)
        self._labels(sc, fleet)
        self._software(sc, fleet)
        self._policies(sc, fleet, team.get("webhook_settings") or {})
        self._reports(sc, fleet)

    # ------------------------------------------------------------ org settings

    def _webhook(self, kind, wh: dict, scope: str, where: str, config_pointer=None):
        wh = {k: v for k, v in wh.items() if k != "policy_ids"}
        if wh.get("destination_url"):
            prefix = "FLEET" if scope == "Global" else f"FLEET_{_envslug(scope)}"
            source = {"type": "config", "pointer": ["webhook_settings", kind, "destination_url"]} if scope == "Global" else \
                {"type": "team_config", "team": self._scope_id(scope), "pointer": ["webhook_settings", kind, "destination_url"]}
            wh["destination_url"] = self.add_env(f"{prefix}_{kind.upper()}_URL", where, f"{kind.replace('_', ' ').capitalize()} destination URL ({scope})", source)
        return wh

    def _scope_id(self, scope_name):
        for s in self.snap["scopes"]:
            if s["name"] == scope_name or (scope_name == "Unassigned" and s["key"] == "unassigned"):
                return s["id"]
        return None

    def _org_settings(self, cfg) -> dict:
        premium = self.flags["premium"]
        o = {}
        info = drop_aliases("org_info", cfg.get("org_info") or {})
        org_info = {"org_name": info.get("org_name", "")}
        for k in ("org_logo_url_dark_mode", "org_logo_url_light_mode", "contact_url"):
            if info.get(k):
                org_info[k] = info[k]
        for mode, body in (self.snap.get("logos") or {}).items():
            ext = _image_ext(body)
            if not ext:
                self.note("gap", "Global", f"org logo ({mode})", "the stored logo isn't PNG, JPEG, WebP, or SVG; the URL is exported as-is")
                continue
            fid = self.add_file("org_logo", "all", f"{mode}{ext}", content=body, exact=True)
            org_info.pop(f"org_logo_url_{mode}_mode", None)
            org_info[f"org_logo_path_{mode}_mode"] = {"$file": fid}
        o["org_info"] = org_info

        ss = drop_aliases("server_settings", cfg.get("server_settings") or {})
        ss["server_url"] = self.add_env("FLEET_URL", "default.yml", "Fleet server URL (also used by CI to reach Fleet)", {"type": "fleet_url"})
        o["server_settings"] = ss
        for k in ("features", "fleet_desktop", "host_expiry_settings", "activity_expiry_settings"):
            if cfg.get(k) is not None:
                o[k] = cfg[k]

        wh = {}
        for k, v in (cfg.get("webhook_settings") or {}).items():
            wh[k] = self._webhook(k, v, "Global", "default.yml") if isinstance(v, dict) else v
        o["webhook_settings"] = wh

        o["integrations"] = self._integrations(cfg.get("integrations") or {})

        sso = dict(cfg.get("sso_settings") or {})
        if sso.get("metadata"):
            sso["metadata"] = self.add_env("FLEET_SSO_METADATA", "default.yml", "IdP metadata XML for admin SSO (uncomment the FLEET_SSO_METADATA line in .github/fleet-gitops/gitops.sh)",
                                           {"type": "config", "pointer": ["sso_settings", "metadata"], "indent": 6})
        if not premium:
            sso.pop("enable_jit_provisioning", None)
            sso.pop("enable_jit_role_sync", None)
        if any(v for v in sso.values()):
            o["sso_settings"] = sso

        mdm = self._org_mdm(cfg)
        if mdm:
            o["mdm"] = mdm
        cas = self._certificate_authorities()
        if cas:
            o["certificate_authorities"] = cas
        creds = self.snap.get("graph_credentials") or []
        if creds:
            o["microsoft_graph_credentials"] = [{"tenant_id": c.get("tenant_id"), "client_id": c.get("client_id")} for c in creds]
            self.note("info", "Global", "microsoft_graph_credentials",
                      "client_secret is left out, so Fleet keeps the secret it has; add it only when you change tenant_id or client_id")
        if not self.exceptions.get("secrets"):
            o["secrets"] = [{"secret": self._enroll_secret_env(None, i, "default.yml")}
                            for i in range(self.snap.get("enroll_secret_count", 0))]
        else:
            self.note("info", "Global", "secrets", "enroll secrets are excepted from GitOps in Fleet settings, so no secrets keys are written")
        rules = cfg.get("yara_rules") or []
        if rules:
            entries = []
            for r in rules:
                fid = self.add_file("yara_rule", "all", r["name"], content=r.get("contents", "").encode(), exact=exact_name_ok(r["name"]),
                                    ext=".yar")
                entries.append({"path": {"$file": fid}})
            o["yara_rules"] = entries
            self.note("gap", "Global", "yara_rules",
                      "fleetctl currently rejects `org_settings.yara_rules[].path` as an unknown key, so the dry run fails until that's fixed; the rule files are exported")
        if (cfg.get("gitops") or {}).get("gitops_mode_enabled"):
            self.note("info", "Global", "gitops", "GitOps mode settings aren't written; fleetctl keeps them as they are when `gitops:` is absent")
        if (cfg.get("smtp_settings") or {}).get("enable_smtp"):
            self.note("info", "Global", "smtp_settings", "SMTP settings aren't written; fleetctl keeps them as they are when `smtp_settings` is absent, and the password can't be read back")
        self.count("Global", "org settings")
        return o

    def _integrations(self, integ: dict) -> dict:
        out = {}
        for k, v in integ.items():
            if k in ("jira", "zendesk") and v:
                items = []
                for i, it in enumerate(v):
                    it = dict(it)
                    it["api_token"] = self.add_env(f"FLEET_{k.upper()}_API_TOKEN" + (f"_{i + 1}" if i else ""), "default.yml",
                                                   f"{k.title()} API token for {it.get('url', '')}", {"type": "mask"})
                    items.append(it)
                out[k] = items
            elif k in ("google_calendar", "google_workspace") and v:
                if k == "google_workspace" and not self.flags["premium"]:
                    continue
                items = []
                for i, it in enumerate(v):
                    it = dict(it)
                    if "api_key_json" in it:
                        it["api_key_json"] = self.add_env(f"FLEET_{k.upper()}_API_KEY_JSON" + (f"_{i + 1}" if i else ""), "default.yml",
                                                          f"{k.replace('_', ' ').title()} service account key JSON for {it.get('domain', '')}", {"type": "mask"})
                    items.append(it)
                out[k] = items
            elif v not in (None, [], {}) or k in ("conditional_access_enabled", "certificates_disable_host_end_user_binding"):
                out[k] = v
        return out

    def _org_mdm(self, cfg) -> dict:
        mdm = cfg.get("mdm") or {}
        out = {}
        if mdm.get("apple_server_url"):
            out["apple_server_url"] = mdm["apple_server_url"]
        eua = {k: v for k, v in (mdm.get("end_user_authentication") or {}).items() if v}
        if eua:
            if eua.get("metadata"):
                eua["metadata"] = self.add_env("FLEET_MDM_SSO_METADATA", "default.yml",
                                               "IdP metadata XML for end-user authentication (uncomment the FLEET_MDM_SSO_METADATA line in .github/fleet-gitops/gitops.sh)",
                                               {"type": "config", "pointer": ["mdm", "end_user_authentication", "metadata"], "indent": 8})
            out["end_user_authentication"] = eua
        if self.flags["premium"]:
            abm = [drop_aliases("apple_business", a) for a in (mdm.get("apple_business") or mdm.get("apple_business_manager") or [])]
            if abm:
                out["apple_business"] = abm
            wae = dict(mdm.get("windows_automatic_enrollment") or {})
            if "default_team" in wae:
                wae.setdefault("default_fleet", wae["default_team"])
                del wae["default_team"]
            if any(wae.values()):
                out["windows_automatic_enrollment"] = wae
            vpp = []
            for t in self.snap.get("vpp_tokens") or []:
                teams = t.get("fleets", t.get("teams"))
                vpp.append({"location": t.get("location"),
                            "fleets": ["All fleets"] if teams == [] else [x.get("name") for x in teams or []]})
            if vpp:
                out["volume_purchasing_program"] = vpp
            for platform, key in (("macos", "end_user_license_agreement"), ("windows", "windows_eula")):
                e = (self.snap.get("eula") or {}).get(platform)
                if e:
                    name = os.path.basename(e["name"] or "")
                    exact = exact_name_ok(name)
                    fid = self.add_file("eula", platform, name if exact else ("eula.pdf" if platform == "macos" else "terms.md"),
                                        content=e["content"], exact=True)
                    out[key] = {"$file": fid}
        return out

    def _certificate_authorities(self) -> dict:
        cas = self.snap.get("certificate_authorities") or {}
        secrets = {"digicert": ["api_token"], "ndes_scep_proxy": ["password"], "custom_scep_proxy": ["challenge"],
                   "hydrant": ["client_secret"], "custom_est_proxy": ["password"], "smallstep": ["username", "password"],
                   "google_cas": ["api_key_json"], "ejbca": ["client_p12", "password"]}
        out = {}
        for kind, val in cas.items():
            if not val:
                continue
            items = val if isinstance(val, list) else [val]
            exported = []
            for i, it in enumerate(items):
                it = dict(it)
                for field in secrets.get(kind, []):
                    label = it.get("name") or kind
                    source = {"type": "mask"} if kind == "ejbca" else {"type": "ca", "kind": kind, "index": i, "field": field}
                    it[field] = self.add_env(f"FLEET_CA_{_envslug(label)}_{field.upper()}", "default.yml",
                                             f"{field} for the {kind} certificate authority {label}", source)
                exported.append(it)
            out[kind] = exported if isinstance(val, list) else exported[0]
            if kind == "ejbca":
                self.note("gap", "Global", "certificate_authorities.ejbca",
                          "the EJBCA client PKCS#12 and its password can't be read back from Fleet; set them before applying")
        return out

    # ------------------------------------------------------------ controls

    def _global_controls(self, mdm: dict) -> dict:
        c = {}
        premium = self.flags["premium"]
        if mdm.get("windows_enabled_and_configured"):
            c["windows_enabled_and_configured"] = True
        if mdm.get("android_enabled_and_configured"):
            c["android_enabled_and_configured"] = True
        # Omitting it means true, so a false has to be written. A server that doesn't return it
        # predates the setting, and so does its fleetctl, which rejects the key.
        if "enable_profile_validation" in mdm:
            c["enable_profile_validation"] = bool(mdm["enable_profile_validation"])
        if not premium:
            return c
        for k in ("windows_migration_enabled", "enable_turn_on_windows_mdm_manually", "apple_require_hardware_attestation",
                  "only_allow_apple_business_enrollment"):
            if mdm.get(k):
                c[k] = True
        mig = mdm.get("macos_migration") or {}
        if mig.get("enable") or mig.get("mode") or mig.get("webhook_url"):
            mig = dict(mig)
            if mig.get("webhook_url"):
                mig["webhook_url"] = self.add_env("FLEET_MACOS_MIGRATION_WEBHOOK_URL", "default.yml", "macOS migration webhook URL",
                                                  {"type": "config", "pointer": ["mdm", "macos_migration", "webhook_url"]})
            c["macos_migration"] = mig
        if mdm.get("windows_enabled_and_configured"):
            for k in ("windows_entra_tenant_ids", "windows_entra_client_ids"):
                if mdm.get(k):
                    c[k] = mdm[k]
        aap = mdm.get("apple_account_provisioning") or {}
        if aap.get("oauth_idp_token_url"):
            c["apple_account_provisioning"] = {
                "oauth_idp_token_url": aap.get("oauth_idp_token_url"),
                "oauth_idp_client_id": aap.get("oauth_idp_client_id"),
                "oauth_idp_client_secret": self.add_env("FLEET_APPLE_ACCOUNT_PROVISIONING_CLIENT_SECRET", "default.yml",
                                                        "OAuth client secret for Apple account provisioning", {"type": "mask"}),
            }
        return c

    def _fleet_controls(self, mdm: dict, sc: dict, fleet: dict) -> dict:
        c: dict = {}
        scope = self._scope_name(fleet) if fleet["key"] != "global" else ("Unassigned" if self.flags["premium"] else "Global")
        premium = self.flags["premium"]
        if premium:
            for k in ("macos_updates", "ios_updates", "ipados_updates"):
                u = mdm.get(k) or {}
                if u.get("minimum_version"):
                    v = {"minimum_version": u["minimum_version"]}
                    if u.get("deadline"):
                        v["deadline"] = u["deadline"]
                    if u.get("deadline_days") is not None:
                        v["deadline_days"] = u["deadline_days"]
                    if k == "macos_updates" and u.get("update_new_hosts") is not None:
                        v["update_new_hosts"] = u["update_new_hosts"]
                    c[k] = v
            wu = mdm.get("windows_updates") or {}
            if wu.get("deadline_days") is not None or wu.get("grace_period_days") is not None:
                c["windows_updates"] = {k: wu.get(k) for k in ("deadline_days", "grace_period_days")}
            if mdm.get("enable_recovery_lock_password"):
                c["enable_recovery_lock_password"] = True
            if mdm.get("name_template"):
                c["name_template"] = mdm["name_template"]
        apple = drop_aliases("settings", mdm.get("apple_settings") or mdm.get("macos_settings") or {})
        windows = drop_aliases("settings", mdm.get("windows_settings") or {})
        linux = mdm.get("linux_settings") or {}
        a, w, lx = {}, {}, {}
        if premium:
            if apple.get("enable_disk_encryption") or mdm.get("enable_disk_encryption"):
                a["enable_disk_encryption"] = True
            if apple.get("enable_escrow_disk_encryption_key"):
                a["enable_escrow_disk_encryption_key"] = True
            if windows.get("enable_disk_encryption"):
                w["enable_disk_encryption"] = True
            if windows.get("require_bitlocker_pin") or mdm.get("windows_require_bitlocker_pin"):
                w["require_bitlocker_pin"] = True
            if windows.get("enable_managed_local_account"):
                w["enable_managed_local_account"] = True
            if linux.get("enable_escrow_disk_encryption_key"):
                lx["enable_escrow_disk_encryption_key"] = True
        if a:
            c["apple_settings"] = a
        if w:
            c["windows_settings"] = w
        if lx:
            c["linux_settings"] = lx

        self._scripts(sc, fleet, scope)
        self._profiles(sc, fleet, scope)
        assets = []
        for asset in sc.get("assets") or []:
            if not exact_name_ok(asset["name"] + ".json"):
                self.note("gap", scope, asset["name"], "Fleet names a DDM asset after its file, and this name can't be a file name; rename the asset in Fleet, then export again")
                continue
            body, escape, refs = self._content_with_env_refs(asset)
            fid = self.add_file("asset", "macos", asset["name"] + ".json", content=body, exact=True, fleet=scope, escape=escape)
            self._record_env_refs(fid, refs, scope)
            assets.append({"path": {"$file": fid}})
            self.count(scope, "DDM assets")
        if assets:
            c.setdefault("apple_settings", {})["assets"] = assets
        certs = []
        for cert in sc.get("android_certificates") or []:
            item = {"name": cert["name"], "certificate_authority_name": cert.get("certificate_authority_name"),
                    "subject_name": cert.get("subject_name")}
            if cert.get("subject_alternative_name"):
                item["subject_alternative_name"] = cert["subject_alternative_name"]
            certs.append(item)
        if certs:
            c.setdefault("android_settings", {})["certificates"] = certs
            self.count(scope, "Android certificates", len(certs))
        if premium and self.flags["apple_mdm"]:
            se = self._setup_experience(mdm, sc, scope)
            if se:
                c["setup_experience"] = se
        return c

    def _setup_experience(self, mdm: dict, sc: dict, scope: str) -> dict:
        api_se = mdm.get("setup_experience") or {}
        if not api_se and mdm.get("macos_setup"):
            api_se = {{"bootstrap_package": "macos_bootstrap_package", "macos_setup_assistant": "apple_setup_assistant",
                       "enable_release_device_manually": "apple_enable_release_device_manually", "script": "macos_script",
                       "manual_agent_install": "macos_manual_agent_install",
                       "enable_managed_local_account": "enable_create_local_admin_account"}.get(k, k): v
                      for k, v in mdm["macos_setup"].items()}
        se = {}
        for k in ("enable_end_user_authentication", "require_all_software_macos", "require_all_software_windows",
                  "apple_enable_release_device_manually", "macos_manual_agent_install", "enable_create_local_admin_account"):
            if api_se.get(k):
                se[k] = True
        # Unset, it follows enable_end_user_authentication, so it's always written.
        se["lock_end_user_info"] = bool(api_se.get("lock_end_user_info"))
        if api_se.get("end_user_local_account_type") not in (None, "", "admin"):
            se["end_user_local_account_type"] = api_se["end_user_local_account_type"]
        setup = sc.get("setup") or {}
        dep = setup.get("enrollment_profile")
        if dep:
            body = dep.get("enrollment_profile") or dep.get("profile")
            name = dep.get("name") or f"{slugify(scope)}.dep.json"
            content = (json.dumps(body, indent=2, ensure_ascii=False) + "\n").encode() if not isinstance(body, str) else body.encode()
            exact = exact_name_ok(name) and name.endswith(".json")
            fid = self.add_file("enrollment_profile", "macos", name if exact else f"{slugify(scope)}.dep.json", content=content, exact=True, fleet=scope)
            se["apple_setup_assistant"] = {"$file": fid}
            self.count(scope, "ADE profile")
        script = setup.get("script")
        if script:
            name = script["name"]
            fid = self.add_file("setup_script", "macos", name if exact_name_ok(name) else f"{slugify(scope)}-setup.sh",
                                content=script["content"], exact=True, fleet=scope)
            se["macos_script"] = {"$file": fid}
            self.count(scope, "setup script")
        if setup.get("bootstrap"):
            self.note("gap", scope, "setup_experience.macos_bootstrap_package",
                      f"bootstrap package {setup['bootstrap'].get('name')!r} stays in Fleet as is (leaving the key out doesn't remove it), "
                      "but Fleet doesn't keep its download URL, so a URL has to be added before this repo can recreate it elsewhere")
        return se

    # ------------------------------------------------------------ scripts and profiles

    def _scripts(self, sc, fleet, scope):
        self._script_ids = getattr(self, "_script_ids", {})
        entries = fleet["entries"].setdefault("controls.scripts", [])
        for s in sc.get("scripts") or []:
            name = s["name"]
            ext = os.path.splitext(name)[1].lower()
            review = None
            if ext == ".ps1":
                platform = "windows"
            else:
                platform, review = _script_platform(name, text_of(s["content"]) or "")
            if not exact_name_ok(name):
                self.note("gap", scope, name, "the script name can't be a file name GitOps can reference with `path:`; rename it in Fleet, then export again")
                continue
            fid = self.add_file("script", platform, name, content=s["content"], exact=True, fleet=scope, review=review)
            names_found = sorted({self._enroll_secret_name(r["source"]["team"], r["source"]["index"])[0] for r in s.get("env_refs") or []})
            if names_found:
                self.note("review", scope, self.files[fid]["path"],
                          f"holds the value of {', '.join(names_found)} as plain text. fleetctl doesn't expand variables in scripts, "
                          "so it's exported as-is; tell the user and suggest a $FLEET_SECRET_ variable instead")
            self._script_ids[(fleet["key"], s["id"])] = fid
            entries.append({"file": fid, "opts": {}})
            self.count(scope, "scripts")

    def _profiles(self, sc, fleet, scope):
        mobile = fleet.get("template") and fleet["template"] == self._mobile_template()
        for p in sc.get("profiles") or []:
            content, name, plat = p["content"], p.get("name", ""), p.get("platform")
            text = text_of(content)
            opts = {}
            for k in ("labels_include_all", "labels_include_any", "labels_exclude_any"):
                if p.get(k):
                    opts[k] = names(p[k])
            if p.get("self_service"):
                opts["self_service"] = True
            if p.get("hidden"):
                opts["hidden"] = True
            if p.get("description"):
                opts["description"] = p["description"]
            review = None
            if plat == "darwin" and content.lstrip().startswith(b"{"):
                kind, ext, section = "declaration", ".json", "controls.apple_settings.configuration_profiles"
                folder_platform = "ios" if mobile else "macos"
            elif plat == "darwin":
                kind, ext, section = "mobileconfig", ".mobileconfig", "controls.apple_settings.configuration_profiles"
                folder_platform = "ios" if mobile else "macos"
                if _payload_display_name(content) != name:
                    opts["name"] = name
            elif plat == "windows":
                kind, ext, section, folder_platform = "windows_profile", ".xml", "controls.windows_settings.configuration_profiles", "windows"
            elif plat == "android":
                kind, ext, section, folder_platform = "android_profile", ".json", "controls.android_settings.configuration_profiles", "android"
            else:
                self.note("gap", scope, name, f"profile platform {plat!r} isn't supported by GitOps; skipped")
                continue
            if plat == "darwin":
                review = ("Apple profiles apply to macOS, iOS, and iPadOS alike; placed under "
                          f"{folder_platform}/ because this fleet looks {'mobile-only' if mobile else 'like computers'}. Move it if another platform fits better.") if mobile else None
            if kind == "mobileconfig":
                filename, exact = slugify(name) + ext, True
            elif exact_name_ok(name + ext):
                filename, exact = name + ext, True
            else:
                filename, exact = slugify(name) + ext, True
                opts["name"] = name
            if text is None:
                self.note("warn", scope, name, "profile isn't UTF-8 text; written as-is without $ escaping")
            body, escape, refs = self._content_with_env_refs(p)
            fid = self.add_file(kind, folder_platform, filename, content=body, exact=exact, fleet=scope,
                                escape=escape, review=review, label=name)
            self._record_env_refs(fid, refs, scope)
            if p.get("activation"):
                afid = self.add_file("activation", folder_platform, filename, content=p["activation"], exact=True, fleet=scope)
                opts["activation"] = {"$file": afid}
            fleet["entries"].setdefault(section, []).append({"file": fid, "opts": opts})
            self.count(scope, {"declaration": "declarations", "mobileconfig": "Apple profiles",
                               "windows_profile": "Windows profiles", "android_profile": "Android profiles"}[kind])

    def _mobile_template(self):
        templates = self.layout.fleet_templates
        by_globs = sorted(templates, key=lambda f: len([g for g in self.layout.globs if g["file"] == f]))
        return by_globs[0] if by_globs else None

    # ------------------------------------------------------------ labels, policies, reports

    def _labels(self, sc, fleet):
        if self.exceptions.get("labels"):
            if sc.get("labels"):
                self.note("info", self._scope_name(fleet), "labels", "labels are managed in the Fleet UI (GitOps exception), so they aren't exported")
            return
        labels = sc.get("labels") or []
        is_global = fleet["key"] == "global"
        directory = self.layout.labels_dir if is_global else f"{self.layout.labels_dir}/{slugify(fleet['sections'].get('name') or fleet['key'], 'fleet')}"
        self._existing_labels = getattr(self, "_existing_labels", set())
        for lbl in labels:
            spec = {"name": lbl["name"]}
            if lbl.get("description"):
                spec["description"] = lbl["description"]
            mtype = lbl.get("label_membership_type") or "dynamic"
            if mtype == "dynamic":
                spec["query"] = lbl.get("query", "")
            spec["label_membership_type"] = mtype
            if mtype == "dynamic" and lbl.get("platform"):
                spec["platform"] = lbl["platform"]
            if mtype == "manual":
                spec["hosts"] = [str(h) for h in (lbl.get("hosts") or [])]
            if mtype == "host_vitals":
                spec["criteria"] = lbl.get("criteria")
            target = self._template_label_file(lbl["name"]) if is_global else None
            if target:
                fid = self.add_file("label", None, os.path.basename(target), data=[spec], exact=True, directory=os.path.dirname(target),
                                    fleet=self._scope_name(fleet), escape=True, label=lbl["name"])
            else:
                fid = self.add_file("label", None, lbl["name"], data=[spec], ext=".yml", directory=directory,
                                    fleet=self._scope_name(fleet), escape=True, label=lbl["name"])
            self._existing_labels.add(self.files[fid]["path"])
            self.count(self._scope_name(fleet), "labels")
            if not is_global:
                fleet["sections"]["labels"] = [{"paths": "../" + directory + "/*.yml"}]

    def _template_label_file(self, label_name):
        for f in self.layout.data["examples"]:
            if f.startswith(self.layout.labels_dir + "/") and f.count("/") == self.layout.labels_dir.count("/") + 1:
                src = self.work / "pristine" / f
                text = (src if src.exists() else self.layout.root / f).read_text(encoding="utf-8")
                m = re.search(r"^- name:\s*(.+?)\s*$", text, re.M)
                if m and m.group(1).strip("\"'") == label_name:
                    return f
        return None

    def _software_index(self, fleet_key):
        return getattr(self, "_sw_index", {}).get(fleet_key, {})

    def _policies(self, sc, fleet, webhook_settings):
        scope = self._scope_name(fleet)
        premium = self.flags["premium"]
        failing = set((webhook_settings.get("failing_policies_webhook") or {}).get("policy_ids") or [])
        sw = self._software_index(fleet["key"])
        for p in sc.get("policies") or []:
            spec = {"name": p["name"]}
            ptype = p.get("type") or "dynamic"
            if ptype != "patch":
                spec["query"] = p.get("query", "")
            for k in ("description", "resolution"):
                if p.get(k):
                    spec[k] = p[k]
            spec["platform"] = p.get("platform", "")
            if premium:
                spec["critical"] = bool(p.get("critical"))
                spec["hidden"] = bool(p.get("hidden"))
            for k in ("calendar_events_enabled", "conditional_access_enabled", "continuous_automations_enabled"):
                spec[k] = bool(p.get(k))
            spec["webhooks_and_tickets_enabled"] = p["id"] in failing
            if ptype != "dynamic":
                spec["type"] = ptype
            if ptype == "patch":
                title = (p.get("patch_software") or {}).get("software_title_id")
                ref = sw.get(title)
                if not ref or ref.get("kind") != "fma":
                    self.note("gap", scope, p["name"], "patch policy points at software that isn't a Fleet-maintained app in this fleet; skipped")
                    continue
                spec["fleet_maintained_app_slug"] = ref["slug"]
                spec["patch_when_closed"] = bool(p.get("patch_when_closed"))
                spec["notify_before_patching"] = bool(p.get("notify_before_patching"))
                if p.get("install_software"):
                    spec["install_software"] = True
            elif p.get("install_software"):
                ref = sw.get(p["install_software"].get("software_title_id"))
                if ref and ref["kind"] == "fma":
                    spec["install_software"] = {"fleet_maintained_app_slug": ref["slug"]}
                elif ref and ref["kind"] == "package":
                    spec["install_software"] = {"package_path": {"$file": ref["file"]}} if ref.get("file") else {"hash_sha256": ref["hash"]}
                elif ref and ref["kind"] == "app_store_app":
                    spec["install_software"] = {"app_store_id": ref["app_store_id"]}
                else:
                    self.note("gap", scope, p["name"], f"install_software points at {p['install_software'].get('name')!r}, which isn't exported for this fleet; automation dropped")
            if p.get("run_script"):
                sfid = getattr(self, "_script_ids", {}).get((self._controls_key(fleet), p["run_script"].get("id")))
                if sfid:
                    spec["run_script"] = {"path": {"$file": sfid}}
                else:
                    self.note("gap", scope, p["name"], f"run_script points at {p['run_script'].get('name')!r}, which isn't exported for this fleet; automation dropped")
            if p.get("resend_configuration_profile"):
                rp = p["resend_configuration_profile"]
                spec["resend_configuration_profile"] = rp.get("name") if isinstance(rp, dict) else rp
            if premium:
                spec.update(scope_labels(p))
            platform = _policy_folder(p.get("platform", ""))
            fid = self.add_file("policy", platform, p["name"], data=[spec], ext=".yml", fleet=scope, escape=True, label=p["name"])
            fleet["entries"].setdefault("policies", []).append({"file": fid, "opts": {}})
            self.count(scope, "policies")

    def _controls_key(self, fleet):
        # Unassigned's scripts live in default.yml's controls (fleetctl allows controls in only one of the two files).
        return "global" if fleet["key"] == "unassigned" else fleet["key"]

    def _reports(self, sc, fleet):
        scope = self._scope_name(fleet)
        for r in sc.get("reports") or []:
            spec = {"name": r["name"]}
            if r.get("description"):
                spec["description"] = r["description"]
            spec["query"] = r.get("query", "")
            spec["platform"] = r.get("platform", "")
            spec["interval"] = r.get("interval", 0)
            for k in ("observer_can_run", "automations_enabled", "discard_data"):
                spec[k] = bool(r.get(k))
            if r.get("min_osquery_version"):
                spec["min_osquery_version"] = r["min_osquery_version"]
            if r.get("logging"):
                spec["logging"] = r["logging"]
            labels = scope_labels(r, ("labels_include_any", "labels_include_all"))
            if not self.flags["premium"]:
                labels.pop("labels_include_all", None)
            spec.update(labels)
            fid = self.add_file("report", _policy_folder(r.get("platform", "")), r["name"], data=[spec], ext=".yml", fleet=scope,
                                escape=True, label=r["name"])
            fleet["entries"].setdefault("reports", []).append({"file": fid, "opts": {}})
            self.count(scope, "reports")

    # ------------------------------------------------------------ software

    def _software(self, sc, fleet):
        scope = self._scope_name(fleet)
        sw = sc.get("software") or {}
        self._sw_index = getattr(self, "_sw_index", {})
        index = self._sw_index.setdefault(fleet["key"], {})
        if self.exceptions.get("software"):
            for t in sw.get("titles", []):
                pkg, app = t.get("software_package") or {}, t.get("app_store_app") or {}
                if pkg.get("fleet_maintained_app_id"):
                    index[t["id"]] = {"kind": "fma", "slug": (t.get("fma") or {}).get("slug")}
                elif pkg:
                    index[t["id"]] = {"kind": "package", "hash": pkg.get("hash_sha256")}
                elif app:
                    index[t["id"]] = {"kind": "app_store_app", "app_store_id": app.get("app_store_id")}
            self.note("info", scope, "software", "software is managed in the Fleet UI (GitOps exception), so no software keys are written")
            return

        native_setup, ipa_setup, app_setup, cross = set(), {}, set(), {}
        for t in sw.get("setup", []):
            pkg, app = t.get("software_package") or {}, t.get("app_store_app") or {}
            if pkg.get("install_during_setup"):
                if pkg.get("name", "").lower().endswith(".ipa"):
                    ipa_setup.setdefault(pkg["name"], []).append(pkg.get("platform"))
                else:
                    native_setup.add(t["id"])
            if app.get("install_during_setup"):
                app_setup.add((app.get("app_store_id"), app.get("platform")))
        for t in sw.get("setup_macos", []):
            pkg = t.get("software_package") or {}
            if pkg.get("platform") and pkg["platform"] != "darwin" and pkg.get("install_during_setup"):
                cross.setdefault(t["id"], []).append("darwin")

        packages, fmas, apps = [], [], []
        seen_ipa = set()
        vpp_count = self._vpp_token_count(sc.get("id"))
        for t in sw.get("titles", []):
            pkg, app = t.get("software_package"), t.get("app_store_app")
            title = t.get("name", "")
            if pkg:
                if pkg.get("name", "").lower().endswith(".ipa"):
                    if pkg["name"] in seen_ipa:
                        continue
                    seen_ipa.add(pkg["name"])
                entry = self._package(t, pkg, fleet, scope, native_setup, ipa_setup, cross, index)
                if entry is None:
                    continue
                (fmas if pkg.get("fleet_maintained_app_id") else packages).append(entry)
            elif app:
                apps.append(self._app_store_app(t, app, scope, app_setup, vpp_count, index))
                self.count(scope, "App Store apps")
            else:
                self.note("warn", scope, title, "software title has neither a package nor an App Store app; skipped")
        soft = {}
        if fmas:
            soft["fleet_maintained_apps"] = fmas
        if apps:
            soft["app_store_apps"] = apps
        fleet["sections"]["software"] = soft
        fleet["entries"]["software.packages"] = packages

    def _package_name_clashes(self) -> set:
        """(platform folder, slug) pairs that several package types share, like a .deb and an .rpm of one app."""
        if self._clashes is None:
            exts: dict = {}
            for sc in self.snap["scopes"]:
                for t in (sc.get("software") or {}).get("titles", []):
                    pkg = t.get("software_package") or {}
                    ext = os.path.splitext(pkg.get("name", ""))[1].lower()
                    if not pkg or pkg.get("fleet_maintained_app_id") or ext in (".sh", ".ps1", ".py"):
                        continue
                    key = (PLATFORM_DIR.get(pkg.get("platform") or "", "all"), slugify(t.get("name", "")))
                    exts.setdefault(key, set()).add(ext)
            self._clashes = {k for k, v in exts.items() if len(v) > 1}
        return self._clashes

    def _vpp_token_count(self, team_id):
        n = 0
        for tok in self.snap.get("vpp_tokens") or []:
            teams = tok.get("fleets", tok.get("teams"))
            if teams == [] or any(x.get("fleet_id", x.get("team_id")) == team_id for x in teams or []):
                n += 1
        return n

    def _setup_keys(self, spec, native: bool, crosses: list, native_platform: str):
        if not crosses:
            if native:
                spec["setup_experience"] = True
            return
        plats = sorted(set(crosses + ([native_platform] if native and native_platform else [])))
        spec["setup_experience_platform"] = ",".join(plats)

    def _package(self, t, pkg, fleet, scope, native_setup, ipa_setup, cross, index):
        title = t.get("name", "")
        platform = PLATFORM_DIR.get(pkg.get("platform") or "", "all")
        fname = pkg.get("name", "")
        ext = os.path.splitext(fname)[1].lower()
        is_script = ext in (".sh", ".ps1", ".py")
        fma = t.get("fma") if pkg.get("fleet_maintained_app_id") else None
        slug = slugify(title)
        files_prefix = slug + ("" if platform in ("macos", "windows", "linux") else f"-{platform}")
        if not fma and not is_script and (platform, slug) in self._package_name_clashes():
            files_prefix += "-" + ext.lstrip(".")
        opts = {}
        if fma:
            index[t["id"]] = {"kind": "fma", "slug": fma.get("slug")}
            opts["slug"] = fma.get("slug")
            if pkg.get("pinned_version"):
                opts["version"] = str(pkg["pinned_version"])
        elif pkg.get("hash_sha256"):
            index[t["id"]] = {"kind": "package", "hash": pkg["hash_sha256"]}

        def side(kind_suffix, body, exe_ext):
            return self.add_file("software_file", platform, f"{files_prefix}-{kind_suffix}{exe_ext}", content=body.encode(), exact=True, fleet=scope)

        def query_file(query):
            return self.add_file("software_file", platform, f"{files_prefix}-pre-install-query.yml", data=[{"query": query}],
                                 exact=True, fleet=scope, escape=True)

        script_ext = ".ps1" if platform == "windows" else ".sh"
        multi = len(t.get("packages") or []) > 1 and not fma

        def common(target, p):
            if p.get("self_service"):
                target["self_service"] = True
            if p.get("categories") is not None and (p.get("categories") or fma):
                target["categories"] = p["categories"]
            target.update(software_labels(p))

        icon = None
        if t.get("icon"):
            icon = self.add_file("icon", "all", f"{files_prefix}-icon.png", content=t["icon"], exact=True, fleet=scope)

        native = t["id"] in native_setup

        if fma:
            entry = dict(opts)
            common(entry, pkg)
            patch = pkg.get("patch_policy") or {}
            managed_query = patch.get("patch_when_closed") or patch.get("notify_before_patching")
            for key, suffix in (("install_script", "install"), ("uninstall_script", "uninstall")):
                state = _fma_script_state(pkg, fma, key)
                if state == "older" and not ({"all", fma.get("slug")} & self.keep_fma_scripts):
                    self._older_fma_scripts.setdefault(scope, []).append(f"{fma.get('slug')} ({suffix})")
                elif state != "default":
                    entry[key] = {"path": {"$file": side(suffix, pkg[key], script_ext)}}
            if pkg.get("post_install_script"):
                entry["post_install_script"] = {"path": {"$file": side("post-install", pkg["post_install_script"], script_ext)}}
            if pkg.get("pre_install_query") and not managed_query:
                entry["pre_install_query"] = {"path": {"$file": query_file(pkg["pre_install_query"])}}
            if t.get("display_name"):
                entry["display_name"] = t["display_name"]
            if icon:
                entry["icon"] = {"path": {"$file": icon}}
            self._setup_keys(entry, native, cross.get(t["id"], []), pkg.get("platform"))
            self.count(scope, "Fleet-maintained apps")
            return entry

        if is_script:
            content = pkg.get("install_script", "").encode()
            entry = {}
            if pkg.get("hash_sha256") and hashlib.sha256(content).hexdigest() == pkg["hash_sha256"] and exact_name_ok(fname):
                fid = self.add_file("software_file", platform, fname, content=content, exact=True, fleet=scope)
                entry["path"] = {"$file": fid}
            else:
                entry["hash_sha256"] = pkg.get("hash_sha256")
                self.note("warn", scope, title, "script package contents don't match its hash; referenced by hash_sha256 instead of a file")
            common(entry, pkg)
            if pkg.get("post_install_script"):
                entry["post_install_script"] = {"path": {"$file": side("post-install", pkg["post_install_script"], script_ext)}}
            if pkg.get("uninstall_script"):
                entry["uninstall_script"] = {"path": {"$file": side("uninstall", pkg["uninstall_script"], script_ext)}}
            if pkg.get("pre_install_query"):
                entry["pre_install_query"] = {"path": {"$file": query_file(pkg["pre_install_query"])}}
            if t.get("display_name"):
                entry["display_name"] = t["display_name"]
            if icon:
                entry["icon"] = {"path": {"$file": icon}}
            self._setup_keys(entry, native, cross.get(t["id"], []), pkg.get("platform"))
            self.count(scope, "script packages")
            return {"file": entry.pop("path")["$file"], "opts": entry} if "path" in entry else {"inline": entry}

        pkgs = t.get("packages") if multi else [pkg]
        items = []
        pkg_dir, _ = self.layout.dir_for("package", platform)
        unpinned = False
        for i, p in enumerate(pkgs):
            prefix = files_prefix if not multi else f"{files_prefix}-{i + 1}"
            item = {}
            if p.get("url"):
                item["url"] = p["url"]
            if p.get("url") and not _url_names_version(p["url"], p.get("version") or pkg.get("version")):
                unpinned = True
                self.note("info", scope, title, "the package URL doesn't name a version, so hash_sha256 isn't pinned and each GitOps run "
                                                "installs what the URL serves then (a pinned hash would fail once the vendor publishes an update)")
            else:
                item["hash_sha256"] = p.get("hash_sha256") or pkg.get("hash_sha256")
            p_ext = os.path.splitext(p.get("name") or fname)[1]
            for key, suffix in (("install_script", "install"), ("uninstall_script", "uninstall"), ("post_install_script", "post-install")):
                if p.get(key) and not is_default_script(key, p_ext, p[key]):
                    item[key] = {"path": {"$file": self.add_file("software_file", platform, f"{prefix}-{suffix}{script_ext}",
                                                                 content=p[key].encode(), exact=True, fleet=scope)}}
            if p.get("pre_install_query"):
                item["pre_install_query"] = {"path": {"$file": self.add_file("software_file", platform, f"{prefix}-pre-install-query.yml",
                                                                             data=[{"query": p["pre_install_query"]}], exact=True, fleet=scope, escape=True)}}
            if multi:
                common(item, p)
            if icon and i == 0:
                item["icon"] = {"path": {"$file": icon}}
            if not item.get("url"):
                self.note("info", scope, title, "package has no URL in Fleet, so it's pinned by hash_sha256 only and can't be re-downloaded on another Fleet instance")
            items.append(item)
        fid = self.add_file("package", platform, f"{files_prefix}.package.yml", data=items, exact=True, fleet=scope, escape=True,
                            directory=pkg_dir, label=title)
        if unpinned and not multi:
            # Without a pinned hash, a policy can't name the package by hash once it updates; package_path finds it by URL.
            index[t["id"]] = {"kind": "package", "hash": pkg.get("hash_sha256"), "file": fid}
        entry = {}
        if not multi:
            common(entry, pkg)
        if t.get("display_name"):
            entry["display_name"] = t["display_name"]
        if fname.lower().endswith(".ipa"):
            plats = sorted(ipa_setup.get(fname, []))
            if plats:
                entry["setup_experience_platform"] = ",".join(plats)
            cfg = pkg.get("configuration")
            if cfg:
                xml = json.loads(cfg) if isinstance(cfg, str) and cfg.startswith('"') else cfg
                if isinstance(xml, str) and xml:
                    entry["configuration"] = {"path": {"$file": self.add_file("app_config", platform, f"{files_prefix}-config.xml",
                                                                             content=xml.encode(), exact=True, fleet=scope)}}
        else:
            self._setup_keys(entry, native, cross.get(t["id"], []), pkg.get("platform"))
        self.count(scope, "custom packages")
        return {"file": fid, "opts": entry}

    def _app_store_app(self, t, app, scope, app_setup, vpp_count, index):
        index[t["id"]] = {"kind": "app_store_app", "app_store_id": app.get("app_store_id")}
        platform = app.get("platform", "")
        prefix = f"{slugify(t.get('name', ''))}-{platform}"
        spec = {"app_store_id": str(app.get("app_store_id")), "platform": platform}
        if app.get("vpp_token") and platform != "android" and vpp_count > 1:
            spec["location"] = app["vpp_token"].get("location")
        if t.get("display_name"):
            spec["display_name"] = t["display_name"]
        if t.get("icon"):
            spec["icon"] = {"path": {"$file": self.add_file("icon", "all", f"{prefix}-icon.png", content=t["icon"], exact=True, fleet=scope)}}
        versions = t.get("app_store_apps") or []
        multi = len(versions) > 1 or (len(versions) == 1 and versions[0].get("name") not in (None, "", "Default version"))
        if not multi:
            v = dict(app)
            for k in ("auto_update_enabled", "auto_update_window_start", "auto_update_window_end"):
                if t.get(k) is not None:
                    v[k] = t[k]
            versions = [v]
        version_specs = []
        used = set()
        for v in versions:
            target = spec if not multi else {"name": v.get("name")}
            vprefix = prefix if not multi else f"{prefix}-{slugify(v.get('name') or '')}"
            if vprefix in used:
                vprefix += f"-{v.get('id')}"
            used.add(vprefix)
            if v.get("self_service"):
                target["self_service"] = True
            if v.get("categories"):
                target["categories"] = v["categories"]
            target.update(software_labels(v))
            cfg = v.get("configuration")
            if cfg not in (None, "", {}):
                if platform in ("ios", "ipados"):
                    xml = cfg if isinstance(cfg, str) else ""
                    if xml:
                        target["configuration"] = {"path": {"$file": self.add_file("app_config", platform, f"{vprefix}-config.xml",
                                                                                  content=xml.encode(), exact=True, fleet=scope)}}
                else:
                    body = (json.dumps(cfg, indent=2, ensure_ascii=False) + "\n").encode()
                    target["configuration"] = {"path": {"$file": self.add_file("app_config", platform or "android", f"{vprefix}-config.json",
                                                                              content=body, exact=True, fleet=scope)}}
            if platform in ("ios", "ipados") and v.get("app_store_id", app.get("app_store_id")):
                for k in ("auto_update_enabled", "auto_update_window_start", "auto_update_window_end"):
                    if v.get(k) is not None:
                        target[k] = v[k]
            if multi:
                version_specs.append(target)
        if multi:
            spec["versions"] = version_specs
        if (app.get("app_store_id"), platform) in app_setup:
            spec["setup_experience"] = True
        return spec

    # ------------------------------------------------------------ cleanup and inventory

    def _examples(self):
        written = {f["path"] for f in self.files.values()}
        for f in self.layout.data["examples"]:
            if f not in written:
                self.remove.append(f)

    def _inventory(self, scopes) -> dict:
        """What Fleet has, by file, read straight from the API data (not from the YAML this
        planner writes), so the reconcile check after placement compares two independent sources.

        Shape: {file: {kind: [[key, {field: value}], ...]}}. Lists, not dicts, so duplicates show.
        """
        inv: dict = {}

        def add(target, kind, key, fields):
            inv.setdefault(target, {}).setdefault(kind, []).append([key, fields])

        premium = self.flags["premium"]
        for key, sc in scopes.items():
            target = "default.yml" if key == "global" else self._fleet_files.get(key, {}).get("file")
            if not target:
                continue
            controls = "default.yml" if key == "unassigned" else target
            sw = self._software_index(key)
            failing = set(((sc.get("team") or self.snap["config"]).get("webhook_settings") or {})
                          .get("failing_policies_webhook", {}).get("policy_ids") or [])
            for p in sc.get("policies") or []:
                f = {k: p.get(k) for k in ("platform", "description", "resolution", "calendar_events_enabled",
                                           "conditional_access_enabled", "continuous_automations_enabled")}
                f["type"] = p.get("type") or "dynamic"
                if f["type"] == "patch":
                    f["fleet_maintained_app_slug"] = (sw.get((p.get("patch_software") or {}).get("software_title_id")) or {}).get("slug")
                else:
                    f["query"] = p.get("query")
                if premium:
                    f.update({"critical": p.get("critical"), "hidden": p.get("hidden")})
                    f.update(scope_labels(p))
                f["webhooks_and_tickets_enabled"] = p["id"] in failing
                f["run_script"] = (p.get("run_script") or {}).get("name")
                add(target, "policies", p["name"], f)
            for r in sc.get("reports") or []:
                f = {k: r.get(k) for k in ("query", "platform", "description", "interval", "observer_can_run", "automations_enabled",
                                           "logging", "discard_data", "min_osquery_version")}
                f.update(scope_labels(r, ("labels_include_any",) + (("labels_include_all",) if premium else ())))
                add(target, "reports", r["name"], f)
            if key != "unassigned" and not self.exceptions.get("labels"):
                for lbl in sc.get("labels") or []:
                    mtype = lbl.get("label_membership_type") or "dynamic"
                    f = {"description": lbl.get("description"), "label_membership_type": mtype}
                    if mtype == "dynamic":
                        f.update({"query": lbl.get("query"), "platform": lbl.get("platform")})
                    elif mtype == "manual":
                        f["hosts"] = [str(h) for h in lbl.get("hosts") or []]
                    else:
                        f["criteria"] = lbl.get("criteria")
                    add(target, "labels", lbl["name"], f)
            for sc_script in sc.get("scripts") or []:
                add(controls, "scripts", sc_script["name"], {"sha256": hashlib.sha256(sc_script["content"]).hexdigest()})
            for prof in sc.get("profiles") or []:
                if prof.get("platform") not in ("darwin", "windows", "android"):
                    continue
                f = {"sha256": hashlib.sha256(prof["content"]).hexdigest(), "self_service": prof.get("self_service"),
                     "hidden": prof.get("hidden"), "description": prof.get("description")}
                f.update(scope_labels(prof, ("labels_include_all", "labels_include_any", "labels_exclude_any")))
                if prof.get("activation"):
                    f["activation_sha256"] = hashlib.sha256(prof["activation"]).hexdigest()
                add(controls, "profiles", prof.get("name"), f)
            for asset in sc.get("assets") or []:
                add(controls, "assets", asset["name"], {"sha256": hashlib.sha256(asset["content"]).hexdigest()})
            if not self.exceptions.get("software") and key != "global":
                seen = set()
                for t in (sc.get("software") or {}).get("titles", []):
                    for ident, f in _software_identities(t):
                        if ident not in seen:
                            seen.add(ident)
                            add(target, "software", ident, f)
        return inv


def _software_identities(t: dict):
    """(identity, settings) for each installable item of a software title, as the reconcile check keys them."""
    pkg, app = t.get("software_package"), t.get("app_store_app")
    if pkg and pkg.get("fleet_maintained_app_id"):
        f = {"self_service": pkg.get("self_service"), "categories": pkg.get("categories")}
        f.update(software_labels(pkg))
        yield f"fma:{(t.get('fma') or {}).get('slug')}", f
    elif pkg:
        is_script = os.path.splitext(pkg.get("name", ""))[1].lower() in (".sh", ".ps1", ".py")
        for p in (t.get("packages") or [pkg]) if len(t.get("packages") or []) > 1 else [pkg]:
            f = {"self_service": p.get("self_service"), "categories": p.get("categories")}
            f.update(software_labels(p))
            # Keyed by URL when there is one: packages whose URL names no version don't pin a hash.
            yield f"package:{(not is_script and p.get('url')) or p.get('hash_sha256') or pkg.get('hash_sha256')}", f
    elif app:
        versions = t.get("app_store_apps") or []
        multi = len(versions) > 1 or (len(versions) == 1 and versions[0].get("name") not in (None, "", "Default version"))
        f = {}
        if not multi:
            f = {"self_service": app.get("self_service"), "categories": app.get("categories")}
            f.update(software_labels(app))
        yield f"app:{app.get('app_store_id')}:{app.get('platform', '')}", f


def _envslug(name: str) -> str:
    return slugify(name, "FLEET").upper().replace("-", "_")


def _policy_folder(platform: str) -> str:
    plats = [p.strip() for p in (platform or "").split(",") if p.strip()]
    if len(plats) == 1 and plats[0] in PLATFORM_DIR:
        return PLATFORM_DIR[plats[0]]
    return "all"


def _script_platform(name: str, text: str) -> tuple[str, str | None]:
    if name.lower().endswith(".ps1"):
        return "windows", None
    lin = len(LINUX_HINTS.findall(name + "\n" + text))
    mac = len(MAC_HINTS.findall(name + "\n" + text))
    if lin > mac:
        return "linux", None
    if mac > lin:
        return "macos", None
    return "macos", "Fleet runs .sh and .py scripts on both macOS and Linux and the contents don't say which this one is for; placed under macos/"


def _fma_script_state(pkg: dict, fma: dict, key: str) -> str:
    """How a Fleet-maintained app's stored script relates to the catalog's: "default", "custom", or "older".

    Fleet records whether an admin edited the script (install_script_edited, uninstall_script_edited)
    and refreshes unedited ones on the app's next update, but the API doesn't return those flags yet,
    so without them a script that differs from the current catalog one is judged by how much it differs.
    """
    stored = pkg.get(key) or ""
    if not stored:
        return "default"
    edited = pkg.get(f"{key}_edited")
    if isinstance(edited, bool):
        return "custom" if edited else "default"
    catalog = fma.get(key) or ""
    if stored.strip() == catalog.strip():
        return "default"
    ratio = difflib.SequenceMatcher(None, stored.splitlines(), catalog.splitlines(), autojunk=False).ratio()
    return "older" if ratio >= FMA_DRIFT_RATIO else "custom"


def _url_names_version(url: str, version: str | None) -> bool:
    """Does the download URL name the package's version (so the bytes behind it shouldn't change)?"""
    if not version:
        return True
    v = version.split(":")[-1]  # Debian epoch
    path = unquote(url)
    return any(c and c in path for c in (v, re.split(r"[-+~]", v)[0]))


def _payload_display_name(content: bytes):
    try:
        return plistlib.loads(content).get("PayloadDisplayName")
    except Exception:  # noqa: BLE001 - a profile with $FLEET_VAR_ in a <data> element isn't a parseable plist
        m = re.findall(rb"<key>PayloadDisplayName</key>\s*<string>(.*?)</string>", content, re.S)
        return m[-1].decode(errors="replace") if m else None


def _image_ext(body: bytes) -> str:
    if body.startswith(b"\x89PNG"):
        return ".png"
    if body[:3] == b"\xff\xd8\xff":
        return ".jpg"
    if body[:4] == b"RIFF" and body[8:12] == b"WEBP":
        return ".webp"
    if b"<svg" in body[:512]:
        return ".svg"
    return ""
