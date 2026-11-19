"""Pull everything the export needs from Fleet, read-only.

Returns one in-memory snapshot. Secret values are dropped here (enroll secrets
become counts, masked values stay masked) so nothing downstream can write
them; the dry run fetches the ones it needs again, in memory. Where an enroll
secret appears inside a profile, DDM asset, script, or software script, the
item records the span or which secret it is, never the value.
"""

from __future__ import annotations

import base64
import binascii
import json
import re
from urllib.parse import quote, quote_plus, urlparse
from xml.sax.saxutils import escape as xml_escape

from fleet_api import API, FleetctlError

F = "/api/latest/fleet"
SETUP_PLATFORMS = "macos,windows,linux,ios,ipados,android"
MIN_SECRET_LEN = 8  # shorter values match ordinary text too often to be replaced safely
# Settings the planner writes key by key from Fleet's data, where a secret value can become a variable.
SETTINGS_PASSED_THROUGH = ("agent_options", "features")
# Package fields the planner writes (into package YAML or as files beside it).
PACKAGE_FIELDS = ("name", "url", "install_script", "uninstall_script", "post_install_script", "pre_install_query",
                  "configuration", "categories")


def _items(d: dict | None, *keys):
    if not d:
        return []
    for k in keys:
        if d.get(k) is not None:
            return d[k]
    return []


def _team_id(obj):
    v = obj.get("fleet_id", obj.get("team_id"))
    return v


def _variants(secret: str) -> list:
    """The forms a secret takes escaped in JSON, a URL, or XML. fleetctl would expand a variable to the plain
    value, so a file holding one of these can't use a variable there."""
    forms = {json.dumps(secret)[1:-1], json.dumps(secret)[1:-1].replace("/", "\\/"), quote(secret, safe=""), quote_plus(secret),
             xml_escape(secret, {'"': "&quot;", "'": "&apos;"})}
    forms.discard(secret)
    return sorted(forms)


def _b64(text: str) -> bytes:
    try:
        return base64.b64decode("".join(text.split()), validate=True)
    except (binascii.Error, ValueError):
        return b""


def _written(item: dict, *keys) -> dict:
    return {k: item[k] for k in keys if k in item}


def _word(secret: str, raw: bool):
    """A short secret as a whole word, for text or (raw) UTF-8 bytes."""
    p = r"(?<![A-Za-z0-9])" + re.escape(secret) + r"(?![A-Za-z0-9])"
    return p.encode() if raw else p


class Collector:
    def __init__(self, api: API, log=print):
        self.api = api
        self.log = log
        self.warnings: list[dict] = []
        self._fma_cache: dict = {}
        self._secret_values: list = []

    def warn(self, scope, item, message):
        w = {"scope": scope, "item": item, "message": message}
        if w not in self.warnings:
            self.warnings.append(w)

    def collect(self) -> dict:
        api = self.api
        me = api.json(f"{F}/me").get("user", {})
        version = api.json(f"{F}/version").get("version", "")
        cfg = api.json(f"{F}/config")
        premium = (cfg.get("license") or {}).get("tier") == "premium"
        mdm = cfg.get("mdm") or {}
        flags = {
            "premium": premium,
            "apple_mdm": bool(mdm.get("enabled_and_configured")),
            "windows_mdm": bool(mdm.get("windows_enabled_and_configured")),
            "android_mdm": bool(mdm.get("android_enabled_and_configured")),
        }
        flags["any_mdm"] = flags["apple_mdm"] or flags["windows_mdm"] or flags["android_mdm"]
        snap = {
            "me": {"email": me.get("email"), "global_role": me.get("global_role")},
            "version": version,
            "config": cfg,
            "flags": flags,
            "gitops": cfg.get("gitops") or {},
        }

        self.log("reading global settings")
        secrets = (api.json(f"{F}/spec/enroll_secret").get("spec") or {}).get("secrets") or []
        snap["enroll_secret_count"] = len(secrets)
        self._secret_values = [(s.get("secret"), {"type": "enroll_secret", "team": None, "index": i}) for i, s in enumerate(secrets)]
        snap["custom_host_vitals"] = self._paged_if_present(f"{F}/custom_host_vitals", "custom_host_vitals")
        snap["certificate_authorities"] = api.json(f"{F}/spec/certificate_authorities").get("certificate_authorities") or {}
        graph = api.maybe_json(f"{F}/microsoft_graph_credentials") if premium else None
        snap["graph_credentials"] = (graph or {}).get("microsoft_graph_credentials") or []
        snap["vpp_tokens"] = api.json(f"{F}/vpp_tokens").get("vpp_tokens") or [] if premium else []
        snap["eula"] = self._eulas(flags) if premium else {}
        snap["logos"] = self._logos(cfg.get("org_info") or {})

        scopes = [{"key": "global", "id": None, "name": "Global"}]
        if premium:
            teams = api.paged(f"{F}/fleets", "fleets")
            scopes.append({"key": "unassigned", "id": 0, "name": "Unassigned"})
            for t in sorted(teams, key=lambda t: t["id"]):
                scopes.append({"key": str(t["id"]), "id": t["id"], "name": t["name"]})
        snap["scopes"] = []
        for sc in scopes:
            self.log(f"reading {sc['name']}")
            snap["scopes"].append(self._scope(sc, flags))
        self._mark_secrets(snap["scopes"], snap)
        self._secret_values = []
        snap["warnings"] = self.warnings
        return snap

    def _mark_secrets(self, scopes, snap: dict | None = None):
        """Mark items holding an enroll secret.

        Profiles and DDM assets get the spans (`env_refs`) of each plain occurrence, which become variables.
        Anything fleetctl sends as-is (scripts, setup scripts, ADE profiles, activations, software, EULAs, YARA
        rules), YAML items (policies, reports, labels, profile metadata), bodies that aren't UTF-8, and encoded
        occurrences (JSON, URL, XML escaping; base64 <data>) only get which secrets (`plain_secret_refs`), and
        the item is left out. A short secret as a whole word does the same (`short_secret_refs`); inside a word
        it's flagged (`short_secret_in_word`) for the planner to warn about if it writes the item. In agent
        options and features, which are written key by key, the paths holding one are listed instead
        (`snap["setting_secret_paths"]`) so the planner can write those values as variables.
        """
        values = [(v, src, _variants(v)) for v, src in self._secret_values if v and len(v) >= MIN_SECRET_LEN]
        short = [(v, src) for v, src in self._secret_values if v and len(v) < MIN_SECRET_LEN]

        def match(text):
            """(plain occurrences' secrets, other occurrences' secrets, short whole-word secrets, short inside a word)."""
            raw = isinstance(text, bytes)
            plain, other = [], []
            blobs = [] if raw else [b for b in (_b64(m) for m in re.findall(r"<data>([A-Za-z0-9+/=\s]+)</data>", text)) if b]
            for v, src, forms in values:
                if raw:
                    if any(v.encode(e) in text for e in ("utf-8", "utf-16-le", "utf-16-be")):
                        other.append(src)
                elif v in text:
                    plain.append(src)
                if not raw and (any(f in text for f in forms) or any(v.encode(e) in b for b in blobs for e in ("utf-8", "utf-16-le"))):
                    other.append(src)
            whole = [src for v, src in short if re.search(_word(v, raw), text)]
            in_word = not whole and any((v.encode() if raw else v) in text for v, _ in short)
            return plain, other, whole, in_word

        def add(owner, key, found):
            if found:
                refs = owner.setdefault(key, [])
                refs.extend(s for s in found if s not in refs)

        def scan(owner, body, spans_key="plain_secret_refs"):
            if isinstance(body, (dict, list)):
                for v in list(body.values() if isinstance(body, dict) else body):
                    scan(owner, v, spans_key)
                return
            if not body or not isinstance(body, (str, bytes)):
                return
            text = body
            if isinstance(body, bytes):
                try:
                    text = body.decode("utf-8")
                except UnicodeDecodeError:
                    spans_key = "plain_secret_refs"  # matched as bytes; it can't take a variable
            plain, other, whole, in_word = match(text)
            if isinstance(body, bytes) and isinstance(text, str) and "\x00" in text:
                other = other + match(body)[1]  # UTF-16 without a byte-order mark can still decode as UTF-8
            add(owner, "short_secret_refs", whole)
            if in_word:
                owner["short_secret_in_word"] = True
            add(owner, "plain_secret_refs", other + (plain if spans_key == "plain_secret_refs" else []))
            if spans_key == "plain_secret_refs" or not plain:
                return
            spans = []
            for v, src, _ in values:
                start = text.find(v)
                while start != -1:
                    spans.append({"start": start, "end": start + len(v), "source": src})
                    start = text.find(v, start + len(v))
            refs, end = [], -1
            for s in sorted(spans, key=lambda s: (s["start"], -s["end"])):
                if s["start"] >= end:
                    refs.append(s)
                    end = s["end"]
            owner[spans_key] = refs

        def setting_paths(value, pointer, out):
            if isinstance(value, dict):
                for k, v in value.items():
                    setting_paths(v, pointer + [k], out)
            elif isinstance(value, list):
                for i, v in enumerate(value):
                    setting_paths(v, pointer + [i], out)
            elif isinstance(value, str) and any(match(value)[:3]):
                out.append(pointer)
            return out

        # Only what the planner writes is scanned, so metadata it drops (authors, timestamps) can't leave an item out.
        for sc in scopes:
            for item in (sc.get("profiles") or []) + (sc.get("assets") or []):
                scan(item, item["content"], "env_refs")
                scan(item, _written(item, "name", "description", "activation"))
            for item in sc.get("scripts") or []:
                scan(item, _written(item, "name", "content"))
            for item in sc.get("policies") or []:
                scan(item, _written(item, "name", "query", "description", "resolution"))
            for item in sc.get("reports") or []:
                scan(item, _written(item, "name", "query", "description"))
            for item in sc.get("labels") or []:
                scan(item, _written(item, "name", "query", "description", "criteria"))
            setup = sc.get("setup") or {}
            if isinstance(setup.get("script"), dict):
                scan(setup["script"], _written(setup["script"], "name", "content"))
            if isinstance(setup.get("enrollment_profile"), dict):
                scan(setup["enrollment_profile"], _written(setup["enrollment_profile"], "name", "enrollment_profile", "profile"))
            for t in (sc.get("software") or {}).get("titles") or []:
                packages = [t.get("software_package") or {}] + (t.get("packages") or [])
                apps = [t.get("app_store_app") or {}] + (t.get("app_store_apps") or [])
                scan(t, [_written(t, "name", "display_name")] + [_written(x, *PACKAGE_FIELDS) for x in packages]
                     + [_written(x, "name", "configuration", "categories") for x in apps])
        if snap is not None:
            cfg = snap.get("config") or {}
            for rule in cfg.get("yara_rules") or []:
                if isinstance(rule, dict):
                    scan(rule, _written(rule, "name", "contents"))
            windows_eula = (snap.get("eula") or {}).get("windows")
            if windows_eula:
                scan(windows_eula, _written(windows_eula, "name", "content"))
            paths = {"global": [p for k in SETTINGS_PASSED_THROUGH for p in setting_paths(cfg.get(k), [k], [])]}
            for sc in scopes:
                team = sc.get("team") or {}
                if sc.get("id"):
                    paths[sc["key"]] = [p for k in SETTINGS_PASSED_THROUGH for p in setting_paths(team.get(k), [k], [])]
            snap["setting_secret_paths"] = {k: v for k, v in paths.items() if v}

    def _paged_if_present(self, path: str, key: str) -> list:
        """A list endpoint that older servers don't have (custom host vitals arrived in 4.90): empty there."""
        try:
            return self.api.paged(path, key)
        except FleetctlError as e:
            if e.status == 404:
                return []
            raise

    # ------------------------------------------------------------ global

    def _eulas(self, flags) -> dict:
        out = {}
        for platform, enabled, params in (("macos", flags["apple_mdm"], {}), ("windows", flags["windows_mdm"], {"platform": "windows"})):
            if not enabled:
                continue
            meta = self.api.maybe_json(f"{F}/setup_experience/eula/metadata", params, missing=(404, 400))
            if not meta:
                continue
            content = self.api.raw(f"{F}/setup_experience/eula/{meta['token']}")
            out[platform] = {"name": meta.get("name", ""), "content": content}
        return out

    def _logos(self, org_info: dict) -> dict:
        out = {}
        for mode, keys in (("light", ("org_logo_url_light_mode", "org_logo_url_light_background")),
                           ("dark", ("org_logo_url_dark_mode", "org_logo_url"))):
            url = next((org_info.get(k) for k in keys if org_info.get(k)), "")
            if url and urlparse(url).path == "/api/latest/fleet/logo":
                body = self.api.maybe_raw(f"{F}/logo", {"mode": mode})
                if body:
                    out[mode] = body
                else:
                    self.warn("Global", f"org logo ({mode})", "the org logo URL points at Fleet but Fleet has no logo stored; the URL is exported as-is")
        return out

    # ------------------------------------------------------------ per scope

    def _scope(self, sc: dict, flags: dict) -> dict:
        api, tid = self.api, sc["id"]
        out = dict(sc)
        is_global, is_team = tid is None, bool(tid)
        fleet_param = {} if tid is None else {"fleet_id": tid}

        if is_team or tid == 0:
            team = api.json(f"{F}/fleets/{tid}")
            team = team.get("fleet") or team.get("team") or {}
            secrets = team.pop("secrets", None) or []
            out["enroll_secret_count"] = len(secrets)
            if is_team:
                self._secret_values += [(s.get("secret"), {"type": "enroll_secret", "team": tid, "index": i}) for i, s in enumerate(secrets)]
            out["team"] = team

        # Global and Unassigned share the global label set; fleets have their own.
        if is_global or is_team:
            specs = api.json(f"{F}/spec/labels", {"fleet_id": tid or 0}).get("specs") or []
            out["labels"] = [s for s in specs if s.get("label_type") in (None, "", "regular")
                             and (_team_id(s) or None) == (tid or None)]

        if is_global:
            pol = api.json(f"{F}/policies")
            out["policies"] = _items(pol, "policies")
        else:
            pol = api.json(f"{F}/fleets/{tid}/policies")
            out["policies"] = [p for p in _items(pol, "policies") if (_team_id(p) or 0) == tid]

        if not tid == 0:
            reps = api.paged(f"{F}/reports", "reports", fleet_param) if tid else api.paged(f"{F}/reports", "reports")
            out["reports"] = [r for r in reps if (_team_id(r) or None) == (tid or None)]

        premium = flags["premium"]
        # On Premium, Global's scripts and profiles are Unassigned's; they're read once, there.
        has_controls = (not premium and is_global) or (premium and not is_global)
        if has_controls:
            out["scripts"] = self._scripts(fleet_param if premium else {})
            if flags["any_mdm"]:
                out["profiles"] = self._profiles(fleet_param if premium else {})
            if premium and flags["apple_mdm"]:
                out["assets"] = self._assets(fleet_param)
            certs = api.maybe_json(f"{F}/certificates", {"fleet_id": tid if tid is not None else ""}, missing=(404, 400))
            out["android_certificates"] = _items(certs, "certificates")
        if premium and not is_global:
            out["software"] = self._software(tid)
            if flags["apple_mdm"]:
                out["setup"] = self._setup(tid)
        return out

    def _scripts(self, params) -> list:
        scripts = self.api.paged(f"{F}/scripts", "scripts", params)

        def body(s):
            return {"id": s["id"], "name": s["name"], "content": self.api.raw(f"{F}/scripts/{s['id']}", {"alt": "media"})}

        return self.api.pmap(body, scripts)

    def _profiles(self, params) -> list:
        try:
            profiles = self.api.paged(f"{F}/configuration_profiles", "profiles", params, per_page=100)
        except FleetctlError as e:
            if e.status == 400:
                return []
            raise

        def body(p):
            p = dict(p)
            p["content"] = self.api.raw(f"{F}/configuration_profiles/{p['profile_uuid']}", {"alt": "media"})
            if p.get("platform") == "darwin" and p["content"].lstrip().startswith(b"{"):
                full = self.api.json(f"{F}/configuration_profiles/{p['profile_uuid']}")
                act = (full.get("profile") or full).get("activation")
                p["activation"] = base64.b64decode(act) if act else b""
            return p

        return self.api.pmap(body, profiles)

    def _assets(self, params) -> list:
        assets = _items(self.api.maybe_json(f"{F}/assets", params, missing=(404, 400)), "assets")

        def body(a):
            return {"name": a["name"], "content": self.api.raw(f"{F}/assets/{a['asset_uuid']}", {"alt": "media"})}

        return self.api.pmap(body, assets)

    def fma(self, fma_id: int) -> dict:
        if fma_id not in self._fma_cache:
            d = self.api.json(f"{F}/software/fleet_maintained_apps/{fma_id}")
            self._fma_cache[fma_id] = d.get("fleet_maintained_app") or d
        return self._fma_cache[fma_id]

    def _software(self, tid: int) -> dict:
        api = self.api
        titles = api.paged(f"{F}/software/titles", "software_titles",
                           {"available_for_install": 1, "fleet_id": tid, "order_key": "name"})

        def detail(t):
            d = api.json(f"{F}/software/titles/{t['id']}", {"fleet_id": tid}).get("software_title") or {}
            icon_url = d.get("icon_url") or ""
            if icon_url.startswith("/api"):
                d["icon"] = api.maybe_raw(f"{F}/software/titles/{t['id']}/icon", {"fleet_id": tid})
            pkg = d.get("software_package") or {}
            if pkg.get("fleet_maintained_app_id"):
                d["fma"] = self.fma(pkg["fleet_maintained_app_id"])
            return d

        details = api.pmap(detail, titles)
        setup = api.paged(f"{F}/setup_experience/software", "software_titles", {"platform": SETUP_PLATFORMS, "fleet_id": tid})
        # Titles a .sh/.py package is selected for on a platform other than its own only show up per target platform.
        setup_macos = api.paged(f"{F}/setup_experience/software", "software_titles", {"platform": "macos", "fleet_id": tid})
        return {"titles": details, "setup": setup, "setup_macos": setup_macos}

    def _setup(self, tid: int) -> dict:
        api = self.api
        params = {"fleet_id": tid} if tid else {}
        out = {}
        dep = api.maybe_json(f"{F}/enrollment_profiles/automatic", params, missing=(404, 400))
        if dep:
            out["enrollment_profile"] = dep
        script = api.maybe_json(f"{F}/setup_experience/script", params, missing=(404, 400))
        if script:
            content = api.raw(f"{F}/setup_experience/script", dict(params, alt="media"))
            out["script"] = {"name": script.get("name", "setup-script.sh"), "content": content}
        boot = api.maybe_json(f"{F}/bootstrap/{tid}/metadata", missing=(404, 400))
        if boot:
            out["bootstrap"] = {"name": boot.get("name"), "sha256": boot.get("sha256")}
        return out
