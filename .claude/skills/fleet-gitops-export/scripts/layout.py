"""What `fleetctl new` generated: fleet files, their globs, folders, and example data.

Destinations are derived from this scan, never hardcoded, so the export follows
the templates when they change: a kind of file goes where a template glob for
its section and platform points, else into an existing platforms/<os>/<kind>
folder, else into a new folder with that name (which the report calls out).
"""

from __future__ import annotations

import json
import os
from pathlib import Path

from yamlio import Doc

PLATFORMS = ("macos", "windows", "linux", "ios", "ipados", "android", "all")

# kind -> (key path in a fleet file, folder name, file extensions)
GLOBBED = {
    "policy": (("policies",), "policies", (".yml",)),
    "report": (("reports",), "reports", (".yml",)),
    "script": (("controls", "scripts"), "scripts", (".sh", ".ps1", ".py")),
    "mobileconfig": (("controls", "apple_settings", "configuration_profiles"), "configuration-profiles", (".mobileconfig",)),
    "declaration": (("controls", "apple_settings", "configuration_profiles"), "declaration-profiles", (".json",)),
    "windows_profile": (("controls", "windows_settings", "configuration_profiles"), "configuration-profiles", (".xml",)),
    "android_profile": (("controls", "android_settings", "configuration_profiles"), "configuration-profiles", (".json",)),
}

# kinds no template glob covers: folder name under platforms/<os>/
FOLDERS = {
    "package": "software",
    "software_file": "software",
    "icon": "icons",
    "enrollment_profile": "enrollment-profiles",
    "app_config": "managed-app-configurations",
    "asset": "assets",
    "activation": "declaration-profiles/activations",
    "setup_script": "setup-experience",
    "eula": "setup-experience",
    "org_logo": "org-logos",
    "yara_rule": "yara-rules",
}

SKIP_DIRS = {".git", ".github", ".agents", ".claude", ".gitlab"}


def norm(p: str) -> str:
    return os.path.normpath(p).replace(os.sep, "/")


class Layout:
    def __init__(self, data: dict):
        self.data = data

    @property
    def root(self) -> Path:
        return Path(self.data["root"])

    @property
    def platform_root(self) -> str:
        return self.data["platform_root"]

    @property
    def labels_dir(self) -> str:
        return self.data["labels_dir"]

    @property
    def fleet_templates(self) -> dict:
        return self.data["fleet_templates"]

    @property
    def globs(self) -> list:
        return self.data["globs"]

    @classmethod
    def scan(cls, root: str) -> "Layout":
        root_p = Path(root).resolve()
        default = root_p / "default.yml"
        if not default.exists():
            raise SystemExit(f"{root}: no default.yml; is this `fleetctl new` output?")

        dirs, files = set(), []
        for dirpath, dirnames, filenames in os.walk(root_p):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            rel = norm(os.path.relpath(dirpath, root_p))
            if rel != ".":
                dirs.add(rel)
            for f in filenames:
                files.append(norm(os.path.join(rel, f)) if rel != "." else f)

        candidates = [d for d in dirs if "/" not in d and sum((d + "/" + p) in dirs for p in PLATFORMS) >= 2]
        platform_root = candidates[0] if candidates else "platforms"

        globs = []
        fleet_templates = {}
        yaml_files = ["default.yml"] + sorted(f for f in files if f.startswith("fleets/") and f.endswith((".yml", ".yaml")))
        for rel in yaml_files:
            doc = Doc((root_p / rel).read_text(encoding="utf-8"))
            base = os.path.dirname(rel)
            for keypath, key, pattern in doc.all_path_globs():
                globs.append({
                    "file": rel,
                    "keypath": list(keypath),
                    "key": key,
                    "pattern": pattern,
                    "root_pattern": norm(os.path.join(base, pattern)),
                })
            if rel != "default.yml":
                name = doc.value_text(["name"]) or ""
                fleet_templates[rel] = {"name": _unquote(name)}

        labels_dir = "labels"
        for g in globs:
            if g["file"] == "default.yml" and g["keypath"] == ["labels"]:
                labels_dir = os.path.dirname(g["root_pattern"])

        examples = sorted(
            f for f in files
            if (f.startswith(platform_root + "/") or f.startswith(labels_dir + "/")) and not f.endswith(".gitkeep")
        )
        return cls({
            "root": str(root_p),
            "platform_root": platform_root,
            "labels_dir": labels_dir,
            "fleet_templates": fleet_templates,
            "globs": globs,
            "dirs": sorted(dirs),
            "files": sorted(files),
            "examples": examples,
        })

    def save(self, path: Path):
        path.write_text(json.dumps(self.data, indent=2, ensure_ascii=False), encoding="utf-8")

    @classmethod
    def load(cls, path: Path) -> "Layout":
        return cls(json.loads(path.read_text(encoding="utf-8")))

    def dir_for(self, kind: str, platform: str) -> tuple[str, str]:
        """(folder relative to the repo root, origin) where origin is glob, folder, or new."""
        if kind in GLOBBED:
            keypath, folder, exts = GLOBBED[kind]
            for g in self.globs:
                parts = g["root_pattern"].split("/")
                if g["keypath"] == list(keypath) and platform in parts and os.path.dirname(g["root_pattern"]).endswith(folder) \
                        and os.path.splitext(g["root_pattern"])[1] in exts:
                    return os.path.dirname(g["root_pattern"]), "glob"
        else:
            folder = FOLDERS[kind]
        candidate = f"{self.platform_root}/{platform}/{folder}"
        if candidate in self.data["dirs"]:
            return candidate, "folder"
        return candidate, "new"

    def fleet_globs(self, fleet_file: str, keypath) -> list:
        return [g for g in self.globs if g["file"] == fleet_file and g["keypath"] == list(keypath)]


def _unquote(s: str) -> str:
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] and s[0] in "\"'":
        if s[0] == '"':
            return json.loads(s)
        return s[1:-1].replace("''", "'")
    return s
