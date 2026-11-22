"""Recognize the install and uninstall scripts Fleet writes for a package on its own.

A package uploaded without scripts gets Fleet's default for its type
(pkg/file/management.go), with $PACKAGE_ID and $UPGRADE_CODE filled in from the
package. Writing those defaults into the repo would pin them, so Fleet's later
improvements to them would never reach that package. The defaults are read from
pkg/file/scripts/ in the Fleet repository: the checkout this skill sits in, or
GitHub when the skill was copied out of it. A script that matches an older
default isn't recognized and is exported as a custom script, as before.
"""

from __future__ import annotations

import re
import urllib.request
from pathlib import Path

SOURCE_URL = "https://raw.githubusercontent.com/fleetdm/fleet/main/pkg/file/scripts/"

# install_pkg_fleetd.sh isn't listed: Fleet installs fleetd's .pkg with it whatever was uploaded, so keeping
# a stored copy changes nothing there, and the API doesn't return the package IDs that tell fleetd apart.
DEFAULTS = {
    "install_script": {"pkg": ["install_pkg.sh"], "msi": ["install_msi.ps1"],
                       "deb": ["install_deb.sh"], "rpm": ["install_rpm.sh"]},
    "uninstall_script": {"pkg": ["uninstall_pkg.sh"], "msi": ["uninstall_msi.ps1", "uninstall_msi_with_upgrade_code.ps1"],
                         "deb": ["uninstall_deb.sh"], "rpm": ["uninstall_rpm.sh"]},
}

# What Fleet substitutes for the placeholders: a quoted ID, or a list of them for .pkg.
_FILLED = r"('[^'\n]*'|\(\n(?:  '[^'\n]*'\n)*\))"
# Mirrors file.PackageIDRegex and file.UpgradeCodeRegex; the suffix character is kept, so it's a lookahead here.
_PLACEHOLDER = re.compile(r'"\$(PACKAGE_ID|UPGRADE_CODE)"(?=\W|$)|\$(PACKAGE_ID|UPGRADE_CODE)(?=\W|$)|"?\$\{(PACKAGE_ID|UPGRADE_CODE)\}"?')

NAMES = sorted({name for by_ext in DEFAULTS.values() for names in by_ext.values() for name in names})

_templates: dict | None = None
_patterns: dict = {}


def _checkout() -> Path | None:
    """pkg/file/scripts/ in the Fleet checkout this skill sits in (<root>/.claude/skills/<skill>/scripts/), if any."""
    parents = Path(__file__).resolve().parents
    d = parents[4] / "pkg" / "file" / "scripts" if len(parents) > 4 else None
    return d if d and all((d / name).is_file() for name in NAMES) else None


def read_defaults(timeout: float = 15) -> tuple[dict, str]:
    """Fleet's default scripts by file name, and where they came from. Raises OSError or ValueError when they can't be read."""
    d = _checkout()
    if d:
        return {name: (d / name).read_text(encoding="utf-8") for name in NAMES}, str(d)
    scripts = {}
    for name in NAMES:
        with urllib.request.urlopen(SOURCE_URL + name, timeout=timeout) as r:  # noqa: S310 - fixed https URL
            scripts[name] = r.read(1 << 20).decode("utf-8")
    return scripts, SOURCE_URL


def use(scripts: dict) -> None:
    """Compare against these scripts. A file name that's missing is never a default."""
    global _templates
    _templates = dict(scripts)
    _patterns.clear()


def _norm(text: str) -> str:
    return "\n".join(line.rstrip() for line in text.replace("\r\n", "\n").split("\n")).strip("\n")


def _pattern(filename: str):
    """(template, regex with one group per filled placeholder, placeholder names in order), or None without that default."""
    if _templates is None:
        use(read_defaults()[0])
    if filename not in _templates:
        return None
    if filename not in _patterns:
        template = _norm(_templates[filename])
        parts, kinds, pos = [], [], 0
        for m in _PLACEHOLDER.finditer(template):
            parts.append(template[pos:m.start()])
            kinds.append(next(g for g in m.groups() if g))
            pos = m.end()
        parts.append(template[pos:])
        _patterns[filename] = (template, re.compile(_FILLED.join(re.escape(p) for p in parts)), kinds)
    return _patterns[filename]


def is_default_script(key: str, extension: str, script: str, ids: dict | None = None) -> bool:
    """Is `script` Fleet's default `key` ("install_script" or "uninstall_script") for this package type?

    `ids` maps a placeholder ("PACKAGE_ID", "UPGRADE_CODE") to the values Fleet would fill in for this
    package. A filled-in value that isn't one of them means an admin retargeted the script, so it isn't
    the default. Placeholders missing from `ids` (the API doesn't return every identifier) accept any value.
    """
    text = _norm(script or "")
    ext = extension.lower().lstrip(".")
    names = DEFAULTS.get(key, {}).get(ext, [])
    if ext == "msi" and key == "uninstall_script":
        # On apply, Fleet fills in the upgrade-code script when the package has an upgrade code, the other one otherwise.
        names = ["uninstall_msi_with_upgrade_code.ps1" if (ids or {}).get("UPGRADE_CODE") else "uninstall_msi.ps1"]
    for filename in names:
        found = _pattern(filename)
        if not found:
            continue
        template, pattern, kinds = found
        if text == template:
            return True
        m = pattern.fullmatch(text)
        if m and all(kind not in (ids or {}) or set(re.findall(r"'([^'\n]*)'", value)) <= set(ids[kind])
                     for kind, value in zip(kinds, m.groups())):
            return True
    return False
