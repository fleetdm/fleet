#!/bin/bash
# Post-install check: fail the install (which triggers the uninstall script)
# unless dpkg reports claude-desktop as fully installed and the binary exists.

set -euo pipefail

PKG="claude-desktop"

STATUS="$(dpkg-query -W -f='${db:Status-Status}' "$PKG" 2>/dev/null || true)"
if [ "$STATUS" != "installed" ]; then
  echo "[verify-claude-desktop] $PKG is not installed (dpkg status: '${STATUS:-missing}')." >&2
  exit 1
fi

if [ ! -x /usr/lib/claude-desktop/claude-desktop ]; then
  echo "[verify-claude-desktop] /usr/lib/claude-desktop/claude-desktop is missing or not executable." >&2
  exit 1
fi

echo "[verify-claude-desktop] $PKG $(dpkg-query -W -f='${Version}' "$PKG") is installed."
