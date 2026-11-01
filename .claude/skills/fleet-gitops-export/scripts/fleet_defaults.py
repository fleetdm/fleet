"""Recognize the install and uninstall scripts Fleet writes for a package on its own.

A package uploaded without scripts gets Fleet's default for its type
(pkg/file/management.go), with $PACKAGE_ID and $UPGRADE_CODE filled in from the
package. Writing those defaults into the repo would pin them, so Fleet's later
improvements to them would never reach that package. The copies in
fleet_default_scripts/ match pkg/file/scripts/; a test inside the Fleet
repository keeps them in sync. A script that matches an older default isn't
recognized and is exported as a custom script, as before.
"""

from __future__ import annotations

import re
from pathlib import Path

_DIR = Path(__file__).resolve().parent / "fleet_default_scripts"

DEFAULTS = {
    "install_script": {"pkg": ["install_pkg.sh", "install_pkg_fleetd.sh"], "msi": ["install_msi.ps1"],
                       "deb": ["install_deb.sh"], "rpm": ["install_rpm.sh"]},
    "uninstall_script": {"pkg": ["uninstall_pkg.sh"], "msi": ["uninstall_msi.ps1", "uninstall_msi_with_upgrade_code.ps1"],
                         "deb": ["uninstall_deb.sh"], "rpm": ["uninstall_rpm.sh"]},
}

# What Fleet substitutes for the placeholders: a quoted ID, or a list of them for .pkg.
_FILLED = r"(?:'[^'\n]*'|\(\n(?:  '[^'\n]*'\n)*\))"
# Mirrors file.PackageIDRegex and file.UpgradeCodeRegex; the suffix character is kept, so it's a lookahead here.
_PLACEHOLDER = re.compile(r'"\$(?:PACKAGE_ID|UPGRADE_CODE)"(?=\W|$)|\$(?:PACKAGE_ID|UPGRADE_CODE)(?=\W|$)|"?\$\{(?:PACKAGE_ID|UPGRADE_CODE)\}"?')

_patterns: dict = {}


def _norm(text: str) -> str:
    return "\n".join(line.rstrip() for line in text.replace("\r\n", "\n").split("\n")).strip("\n")


def _pattern(filename: str):
    if filename not in _patterns:
        template = _norm((_DIR / filename).read_text(encoding="utf-8"))
        parts = _PLACEHOLDER.split(template)
        _patterns[filename] = (template, re.compile(_FILLED.join(re.escape(p) for p in parts)))
    return _patterns[filename]


def is_default_script(key: str, extension: str, script: str) -> bool:
    """Is `script` Fleet's default `key` ("install_script" or "uninstall_script") for this package type?"""
    text = _norm(script or "")
    for filename in DEFAULTS.get(key, {}).get(extension.lower().lstrip("."), []):
        template, pattern = _pattern(filename)
        if text == template or pattern.fullmatch(text):
            return True
    return False
