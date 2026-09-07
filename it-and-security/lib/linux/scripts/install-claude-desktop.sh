#!/bin/bash
# Installs the newest Claude Desktop from Anthropic's apt package pool on
# Debian-based hosts (amd64 or arm64).

set -euo pipefail

PKG="claude-desktop"
REPO="https://downloads.claude.ai/claude-desktop/apt/stable"
export DEBIAN_FRONTEND=noninteractive
APT_OPTS=(-y -o DPkg::Lock::Timeout=300)

if [ "$(id -u)" -ne 0 ]; then
  echo "[install-claude-desktop] This script must run as root." >&2
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1 || ! command -v dpkg >/dev/null 2>&1; then
  echo "[install-claude-desktop] apt-get/dpkg not found. Claude Desktop is only published for Debian-based Linux." >&2
  exit 1
fi

ARCH="$(dpkg --print-architecture)"
case "$ARCH" in
  amd64|arm64) ;;
  *)
    echo "[install-claude-desktop] Unsupported architecture '$ARCH'. Anthropic publishes amd64 and arm64 only." >&2
    exit 1
    ;;
esac

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Pick the newest package for this architecture from the repo index and keep
# its published SHA256 so the download can be verified.
curl -fsSL --retry 3 "$REPO/dists/stable/main/binary-$ARCH/Packages" -o "$TMP/Packages"
read -r VERSION FILENAME SHA256 < <(
  awk -v RS= -F'\n' '{
    v=""; f=""; s="";
    for (i = 1; i <= NF; i++) {
      if ($i ~ /^Version: /) v = substr($i, 10);
      else if ($i ~ /^Filename: /) f = substr($i, 11);
      else if ($i ~ /^SHA256: /) s = substr($i, 9);
    }
    if (v != "" && f != "" && s != "") print v, f, s;
  }' "$TMP/Packages" | sort -V | tail -n 1
) || true

if [ -z "${VERSION:-}" ] || [ -z "${FILENAME:-}" ] || [ -z "${SHA256:-}" ]; then
  echo "[install-claude-desktop] Could not find a $PKG package for $ARCH in the repository index." >&2
  exit 1
fi

INSTALLED="$(dpkg-query -W -f='${db:Status-Status} ${Version}' "$PKG" 2>/dev/null || true)"
if [ "$INSTALLED" = "installed $VERSION" ]; then
  echo "[install-claude-desktop] $PKG $VERSION is already installed."
  exit 0
fi

echo "[install-claude-desktop] Downloading $PKG $VERSION ($ARCH)..."
curl -fsSL --retry 3 "$REPO/$FILENAME" -o "$TMP/$PKG.deb"
echo "$SHA256  $TMP/$PKG.deb" | sha256sum -c --quiet

# Refresh distro indexes so apt can resolve the package's dependencies on a
# host that has never run apt-get update; stale indexes are not fatal.
if ! apt-get update; then
  echo "[install-claude-desktop] Warning: apt-get update failed; continuing with cached package indexes." >&2
fi

echo "[install-claude-desktop] Installing $PKG $VERSION..."
apt-get install "${APT_OPTS[@]}" "$TMP/$PKG.deb"

if [ ! -s /etc/apt/sources.list.d/claude-desktop.list ]; then
  echo "[install-claude-desktop] Note: Anthropic's apt repository was not registered (CLAUDE_DESKTOP_ADD_REPO may be false). Updates will re-download the .deb instead."
fi

echo "[install-claude-desktop] Installed $PKG $VERSION."
