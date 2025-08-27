#!/bin/sh
set -e

# Create a temp mountpoint to avoid issues with spaces in volume name
MOUNT_POINT="$(mktemp -d /tmp/omnissa_mount_XXXXXX)"

cleanup() {
  /usr/bin/hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Mount DMG to our custom mountpoint
/usr/bin/hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"

# Find the pkg inside
PKG="$(/usr/bin/find "$MOUNT_POINT" -type f -name '*.pkg' -print -quit)"
[ -n "$PKG" ] || { echo "omnissa pkg not found"; exit 1; }

# Clear quarantine (helps if Gatekeeper blocks it)
xattr -dr com.apple.quarantine "$PKG" || true

# Run installer
/usr/sbin/installer -pkg "$PKG" -target /

echo "omnissa horizon client installed"
