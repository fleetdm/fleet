"""Read-only access to Fleet through fleetctl.

Every call goes through `fleetctl`, so this code never sees the API token:
fleetctl reads it from its own config (and --config/--context pick which one).
The only subcommands allowed are `api` without -X (always GET), `new`,
`--version`, `config get address`, and `gitops` with --dry-run hardcoded.
Anything else raises before a process starts.
"""

from __future__ import annotations

import json
import re
import subprocess
from concurrent.futures import ThreadPoolExecutor

# GET endpoints this skill reads. Each is one the generate-gitops command
# relies on, or the documented download (alt=media) form of one.
ALLOWED_PATHS = [
    r"/api/latest/fleet/me",
    r"/api/latest/fleet/version",
    r"/api/latest/fleet/config",
    r"/api/latest/fleet/spec/enroll_secret",
    r"/api/latest/fleet/spec/labels",
    r"/api/latest/fleet/spec/certificate_authorities",
    r"/api/latest/fleet/fleets",
    r"/api/latest/fleet/fleets/\d+",
    r"/api/latest/fleet/fleets/\d+/policies",
    r"/api/latest/fleet/policies",
    r"/api/latest/fleet/reports",
    r"/api/latest/fleet/scripts",
    r"/api/latest/fleet/scripts/\d+",
    r"/api/latest/fleet/configuration_profiles",
    r"/api/latest/fleet/mdm/profiles/[A-Za-z0-9_\-]+",
    r"/api/latest/fleet/assets",
    r"/api/latest/fleet/assets/[A-Za-z0-9_\-]+",
    r"/api/latest/fleet/certificates",
    r"/api/latest/fleet/custom_host_vitals",
    r"/api/latest/fleet/microsoft_graph_credentials",
    r"/api/latest/fleet/vpp_tokens",
    r"/api/latest/fleet/logo",
    r"/api/latest/fleet/setup_experience/eula/metadata",
    r"/api/latest/fleet/setup_experience/eula/[A-Za-z0-9_\-]+",
    r"/api/latest/fleet/setup_experience/software",
    r"/api/latest/fleet/setup_experience/script",
    r"/api/latest/fleet/enrollment_profiles/automatic",
    r"/api/latest/fleet/mdm/bootstrap/\d+/metadata",
    r"/api/latest/fleet/software/titles",
    r"/api/latest/fleet/software/titles/\d+",
    r"/api/latest/fleet/software/titles/\d+/icon",
    r"/api/latest/fleet/software/fleet_maintained_apps",
    r"/api/latest/fleet/software/fleet_maintained_apps/\d+",
]
_ALLOWED = [re.compile(p + r"$") for p in ALLOWED_PATHS]

_STATUS = re.compile(r"non 2XX return of (\d+)")


class FleetctlError(Exception):
    def __init__(self, message: str, status: int | None = None):
        super().__init__(message)
        self.status = status


class Fleetctl:
    def __init__(self, binary: str = "fleetctl", config: str | None = None, context: str | None = None):
        self.binary = binary
        self.config = config
        self.context = context

    def _conn_flags(self) -> list[str]:
        flags = []
        if self.config:
            flags += ["--config", self.config]
        if self.context:
            flags += ["--context", self.context]
        return flags

    def _run(self, args: list[str], env: dict | None = None, cwd: str | None = None, timeout: int = 300):
        return subprocess.run([self.binary] + args, capture_output=True, env=env, cwd=cwd, timeout=timeout)

    def version(self) -> str:
        r = self._run(["--version"], timeout=60)
        if r.returncode:
            raise FleetctlError(r.stderr.decode(errors="replace").strip() or "fleetctl --version failed")
        m = re.search(r"version\s+(\S+)", r.stdout.decode(errors="replace"))
        return m.group(1) if m else "unknown"

    def has_new(self) -> bool:
        r = self._run(["new", "--help"], timeout=60)
        return r.returncode == 0 and b"--org-name" in r.stdout

    def address(self) -> str:
        r = self._run(["config", "get"] + self._conn_flags() + ["address"], timeout=60)
        out = r.stdout.decode(errors="replace")
        m = re.search(r"=>\s*(\S+)", out)
        if r.returncode or not m:
            raise FleetctlError("couldn't read the address from fleetctl's config; run `fleetctl login` or `fleetctl config set --address`")
        return m.group(1)

    def new(self, directory: str, org_name: str, force: bool = False) -> str:
        args = ["new", "--dir", directory, "--org-name", org_name]
        if force:
            args.append("--force")
        r = self._run(args, timeout=120)
        if r.returncode:
            raise FleetctlError(r.stderr.decode(errors="replace").strip() or r.stdout.decode(errors="replace").strip())
        return r.stdout.decode(errors="replace")

    def gitops_dry_run(self, files: list[str], cwd: str, env: dict, delete_other_fleets: bool = True):
        args = ["gitops"] + self._conn_flags()
        for f in files:
            args += ["-f", f]
        if delete_other_fleets:
            args.append("--delete-other-fleets")
        args.append("--dry-run")
        return self._run(args, env=env, cwd=cwd, timeout=1800)

    def get(self, path: str, params: dict | None = None) -> bytes:
        if not any(p.match(path) for p in _ALLOWED):
            raise FleetctlError(f"refusing to call {path}: not an allowlisted read-only endpoint")
        args = ["api"] + self._conn_flags()
        for k, v in (params or {}).items():
            args += ["-F", f"{k}={v}"]
        args.append(path)
        r = self._run(args)
        if r.returncode:
            err = r.stderr.decode(errors="replace").strip()
            m = _STATUS.search(err)
            raise FleetctlError(f"GET {path}: {err}", int(m.group(1)) if m else None)
        return r.stdout


class API:
    """JSON helpers over Fleetctl.get, with pagination and parallel fetches."""

    def __init__(self, fleetctl: Fleetctl, workers: int = 8):
        self.fc = fleetctl
        self.workers = workers

    def raw(self, path: str, params: dict | None = None) -> bytes:
        return self.fc.get(path, params)

    def json(self, path: str, params: dict | None = None) -> dict:
        body = self.fc.get(path, params)
        return json.loads(body) if body.strip() else {}

    def maybe_json(self, path: str, params: dict | None = None, missing=(404,)):
        try:
            return self.json(path, params)
        except FleetctlError as e:
            if e.status in missing:
                return None
            raise

    def maybe_raw(self, path: str, params: dict | None = None, missing=(404,)):
        try:
            return self.raw(path, params)
        except FleetctlError as e:
            if e.status in missing:
                return None
            raise

    def paged(self, path: str, key: str, params: dict | None = None, per_page: int = 500) -> list:
        items: list = []
        page = 0
        while True:
            p = dict(params or {})
            p.update({"per_page": per_page, "page": page})
            d = self.json(path, p)
            batch = d.get(key) or []
            items.extend(batch)
            if "meta" in d:
                if not (d["meta"] or {}).get("has_next_results"):
                    break
            elif len(batch) < per_page:
                break
            if not batch or page >= 1000:
                break
            page += 1
        return items

    def pmap(self, fn, items):
        items = list(items)
        if len(items) <= 1:
            return [fn(i) for i in items]
        with ThreadPoolExecutor(max_workers=self.workers) as ex:
            return list(ex.map(fn, items))
