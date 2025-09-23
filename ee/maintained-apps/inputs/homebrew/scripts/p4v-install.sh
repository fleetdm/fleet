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

APPDIR="/Applications"

quit_application "com.perforce.p4v"
quit_application "com.perforce.p4merge"
quit_application "com.perforce.p4admin"

MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\//{print $3; exit}')"
[[ -n "$MOUNT_POINT" ]] || { echo "failed to mount dmg"; exit 1; }

for app in p4v.app p4merge.app p4admin.app; do
  if [[ -d "$MOUNT_POINT/$app" ]]; then
    rm -rf "$APPDIR/$app" >/dev/null 2>&1 || true
    ditto "$MOUNT_POINT/$app" "$APPDIR/$app" >/dev/null 2>&1
  fi
done

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true

echo "p4v installed"
