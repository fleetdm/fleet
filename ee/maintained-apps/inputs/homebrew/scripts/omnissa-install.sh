#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10
  if ! osascript -e "application id \"$bundle_id\" is running" >/dev/null 2>&1; then return; fi
  local console_user; console_user=$(stat -f "%Su" /dev/console)
  if [ "$EUID" -eq 0 ] && [ "$console_user" = "root" ]; then echo "Skipping quit for '$bundle_id'."; return; fi
  echo "Quitting '$bundle_id'..."
  SECONDS=0
  while [ "$SECONDS" -lt "$timeout_duration" ]; do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1 || true
    if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then echo "'$bundle_id' quit successfully."; return; fi
    sleep 1
  done
  echo "'$bundle_id' did not quit."
}

[ -n "$INSTALLER_PATH" ] && [ -f "$INSTALLER_PATH" ] || { echo "missing installer"; exit 1; }

quit_application "com.omnissa.horizon.client.mac"

# Mount to a known temp directory to avoid space-in-path issues
MOUNT_POINT="$(mktemp -d "/tmp/omnissa_hzn.XXXXXX")"
# hdiutil needs the directory to exist; mktemp already created it
if ! hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null 2>&1; then
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "failed to mount dmg"
  exit 1
fi

# Look for a pkg (or mpkg) up to a few levels deep
PKG="$(/usr/bin/find "$MOUNT_POINT" -maxdepth 5 -type f \( -name "*.pkg" -o -name "*.mpkg" \) -print -quit)"

if [ ! -f "$PKG" ]; then
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "pkg not found"
  exit 1
fi

/usr/sbin/installer -pkg "$PKG" -target / >/dev/null 2>&1

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true

echo "omnissa horizon client installed"
