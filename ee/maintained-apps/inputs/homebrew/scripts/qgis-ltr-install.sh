#!/bin/bash

# QGIS LTR is closed if it's running, and unsaved project changes are lost. It
# installs as QGIS-LTR.app so it can sit alongside QGIS, which is left alone.

APPDIR="/Applications"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
APP_PATH="$APPDIR/QGIS-LTR.app"

# QGIS and QGIS LTR share a bundle identifier, so match processes by path.
quit_app_at_path() {
  local app_path="$1"
  local pattern="^${app_path}/Contents/MacOS/"
  pgrep -f "$pattern" >/dev/null || return 0
  echo "Closing $app_path..."
  pkill -TERM -f "$pattern"
  for _ in {1..10}; do
    pgrep -f "$pattern" >/dev/null || return 0
    sleep 1
  done
  echo "$app_path did not close; forcing it to quit."
  pkill -KILL -f "$pattern"
}

MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null || exit 1
trap 'hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true' EXIT

if [[ ! -d "$MOUNT_POINT/QGIS.app" ]]; then
  echo "No QGIS.app found in the disk image."
  exit 1
fi

quit_app_at_path "$APP_PATH"

sudo rm -rf "$TMPDIR/QGIS-LTR.app.bkp"
if [[ -d "$APP_PATH" ]]; then
  sudo mv "$APP_PATH" "$TMPDIR/QGIS-LTR.app.bkp" || exit $?
fi
if ! sudo cp -R "$MOUNT_POINT/QGIS.app" "$APP_PATH"; then
  # remove the partial copy so a failed install isn't inventoried as the new
  # version, then restore the previous version if there was one
  sudo rm -rf "$APP_PATH"
  if [[ -d "$TMPDIR/QGIS-LTR.app.bkp" ]]; then
    sudo mv "$TMPDIR/QGIS-LTR.app.bkp" "$APP_PATH"
  fi
  exit 1
fi
sudo rm -rf "$TMPDIR/QGIS-LTR.app.bkp"

echo "Installed $APP_PATH"
