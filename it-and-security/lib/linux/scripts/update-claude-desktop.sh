#!/bin/bash
# Upgrades Claude Desktop from Anthropic's apt repository (policy remediation).
#
# An in-place upgrade while the app is open leaves the old main process spawning
# helpers from the new binary, so the upgrade is deferred (exit 0) while Claude is
# running and proceeds once it's closed, or after MAX_DEFER_DAYS regardless.
# Falls back to the .deb from the package pool if the repo entry is missing.

set -euo pipefail

PKG="claude-desktop"
REPO="https://downloads.claude.ai/claude-desktop/apt/stable"
LIST="/etc/apt/sources.list.d/claude-desktop.list"
MARKER="/var/lib/fleet/claude-desktop-update-deferred"
MAX_DEFER_DAYS=7
export DEBIAN_FRONTEND=noninteractive
APT_OPTS=(-y -o DPkg::Lock::Timeout=300)

if [ "$(id -u)" -ne 0 ]; then
  echo "[update-claude-desktop] This script must run as root." >&2
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1 || ! command -v dpkg >/dev/null 2>&1; then
  echo "[update-claude-desktop] apt-get/dpkg not found. Claude Desktop is only published for Debian-based Linux." >&2
  exit 1
fi

if pgrep -x "$PKG" >/dev/null 2>&1; then
  if [ ! -e "$MARKER" ]; then
    mkdir -p "$(dirname "$MARKER")"
    touch "$MARKER"
  fi
  deferred_for=$(( ( $(date +%s) - $(stat -c %Y "$MARKER") ) / 86400 ))
  if [ "$deferred_for" -lt "$MAX_DEFER_DAYS" ]; then
    echo "[update-claude-desktop] Claude is open; deferring the upgrade (deferred for $deferred_for of $MAX_DEFER_DAYS days). Will retry on the next policy run."
    exit 0
  fi
  echo "[update-claude-desktop] Claude is open but the upgrade has been deferred for $deferred_for days; upgrading anyway. The user should relaunch Claude afterwards."
fi

before="$(dpkg-query -W -f='${Version}' "$PKG" 2>/dev/null || echo none)"

install_from_pool() {
  local arch tmp version filename sha256
  arch="$(dpkg --print-architecture)"
  case "$arch" in
    amd64|arm64) ;;
    *)
      echo "[update-claude-desktop] Unsupported architecture '$arch'." >&2
      return 1
      ;;
  esac

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  curl -fsSL --retry 3 "$REPO/dists/stable/main/binary-$arch/Packages" -o "$tmp/Packages"
  read -r version filename sha256 < <(
    awk -v RS= -F'\n' '{
      v=""; f=""; s="";
      for (i = 1; i <= NF; i++) {
        if ($i ~ /^Version: /) v = substr($i, 10);
        else if ($i ~ /^Filename: /) f = substr($i, 11);
        else if ($i ~ /^SHA256: /) s = substr($i, 9);
      }
      if (v != "" && f != "" && s != "") print v, f, s;
    }' "$tmp/Packages" | sort -V | tail -n 1
  ) || true
  if [ -z "${version:-}" ] || [ -z "${filename:-}" ] || [ -z "${sha256:-}" ]; then
    echo "[update-claude-desktop] Could not find a $PKG package for $arch in the repository index." >&2
    return 1
  fi

  echo "[update-claude-desktop] Downloading $PKG $version ($arch)..."
  curl -fsSL --retry 3 "$REPO/$filename" -o "$tmp/$PKG.deb"
  echo "$sha256  $tmp/$PKG.deb" | sha256sum -c --quiet
  apt-get install "${APT_OPTS[@]}" "$tmp/$PKG.deb"
}

if [ -s "$LIST" ]; then
  echo "[update-claude-desktop] Refreshing Anthropic's apt repository index..."
  # Limit the refresh to Anthropic's repo so this stays fast and doesn't
  # depend on every other configured repository being reachable.
  apt-get update \
    -o Dir::Etc::SourceList="$LIST" \
    -o Dir::Etc::SourceParts=/dev/null \
    -o APT::Get::List-Cleanup=0
  apt-get install "${APT_OPTS[@]}" "$PKG"
else
  echo "[update-claude-desktop] $LIST not found; installing directly from the package pool."
  install_from_pool
fi

rm -f "$MARKER"
after="$(dpkg-query -W -f='${Version}' "$PKG" 2>/dev/null || echo none)"
echo "[update-claude-desktop] $PKG: $before -> $after"
