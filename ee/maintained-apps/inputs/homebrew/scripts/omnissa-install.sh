#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10
  if ! osascript -e "application id \"$bundle_id\" is running" >/dev/null 2>&1; then return; fi
  local console_user; console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then echo "Skipping quit for '$bundle_id'."; return; fi
  echo "Quitting '$bundle_id'..."
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1 || true
    if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then echo "'$bundle_id' quit successfully."; return; fi
    sleep 1
  done
  echo "'$bundle_id' did not quit."
}

[[ -n "$INSTALLER_PATH" && -f "$INSTALLER_PATH" ]] || { echo "missing installer"; exit 1; }

quit_application "com.omnissa.horizon.client.mac"

MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\//{print $3; exit}')"
[[ -n "$MOUNT_POINT" ]] || { echo "failed to mount dmg"; exit 1; }

PKG="$(/usr/bin/find "$MOUNT_POINT" -maxdepth 2 -type f -name "*.pkg" -print -quit)"
[[ -f "$PKG" ]] || { echo "pkg not found"; hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1; exit 1; }

/usr/sbin/installer -pkg "$PKG" -target / >/dev/null

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true

echo "omnissa horizon client installed"
