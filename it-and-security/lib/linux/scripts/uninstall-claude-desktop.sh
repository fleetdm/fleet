#!/bin/bash
# Removes Claude Desktop from Debian-based hosts. The package's postrm also
# removes the apt repository entry, signing key, and unattended-upgrades
# snippet that its postinst registered. Per-user data under ~/.config is left
# in place.

set -euo pipefail

PKG="claude-desktop"
export DEBIAN_FRONTEND=noninteractive

if [ "$(id -u)" -ne 0 ]; then
  echo "[uninstall-claude-desktop] This script must run as root." >&2
  exit 1
fi

STATUS="$(dpkg-query -W -f='${db:Status-Status}' "$PKG" 2>/dev/null || true)"
if [ "$STATUS" != "installed" ]; then
  echo "[uninstall-claude-desktop] $PKG is not installed."
  exit 0
fi

pkill -x "$PKG" 2>/dev/null || true

echo "[uninstall-claude-desktop] Removing $PKG..."
apt-get remove -y -o DPkg::Lock::Timeout=300 "$PKG"

echo "[uninstall-claude-desktop] Removed $PKG."
