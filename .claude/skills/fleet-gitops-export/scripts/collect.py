"""Pull everything the export needs from Fleet, read-only.

Returns one in-memory snapshot. Secret values are dropped here (enroll secrets
become counts, masked values stay masked) so nothing downstream can write
them; the dry run fetches the ones it needs again, in memory.
"""

from __future__ import annotations

import base64
from urllib.parse import urlparse

from fleet_api import API, FleetctlError

F = "/api/latest/fleet"
SETUP_PLATFORMS = "macos,windows,linux,ios,ipados,android"


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


class Collector:
    def __init__(self, api: API, log=print):
        self.api = api
        self.log = log
        self.warnings: list[dict] = []
        self._fma_cache: dict = {}

    def warn(self, scope, item, message):
        self.warnings.append({"scope": scope, "item": item, "message": message})

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
        snap["enroll_secret_count"] = len((api.json(f"{F}/spec/enroll_secret").get("spec") or {}).get("secrets") or [])
        snap["custom_host_vitals"] = api.paged(f"{F}/custom_host_vitals", "custom_host_vitals")
        snap["certificate_authorities"] = api.json(f"{F}/spec/certificate_authorities").get("certificate_authorities") or {}
        snap["graph_credentials"] = api.json(f"{F}/microsoft_graph_credentials").get("microsoft_graph_credentials") or [] if premium else []
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
        snap["warnings"] = self.warnings
        return snap

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
                    self.warn("global", f"org logo ({mode})", "the org logo URL points at Fleet but Fleet has no logo stored; the URL is exported as-is")
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
            out["enroll_secret_count"] = len(team.pop("secrets", None) or [])
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
            p["content"] = self.api.raw(f"{F}/mdm/profiles/{p['profile_uuid']}", {"alt": "media"})
            if p.get("platform") == "darwin" and p["content"].lstrip().startswith(b"{"):
                full = self.api.json(f"{F}/mdm/profiles/{p['profile_uuid']}")
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
        boot = api.maybe_json(f"{F}/mdm/bootstrap/{tid}/metadata", missing=(404, 400))
        if boot:
            out["bootstrap"] = {"name": boot.get("name"), "sha256": boot.get("sha256")}
        return out
