#!/bin/bash

# QGIS is closed if it's running, and unsaved project changes are lost. Each
# release installs as its own QGIS-final-<version>.app, so earlier releases are
# removed once the new one is in place. QGIS LTR (QGIS-LTR.app) is left alone.

APPDIR="/Applications"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")

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

NEW_APP=$(find "$MOUNT_POINT" -maxdepth 1 -name 'QGIS-final-*.app' -print -quit)
if [[ -z "$NEW_APP" ]]; then
  echo "No QGIS-final-*.app found in the disk image."
  exit 1
fi
APP_NAME=$(basename "$NEW_APP")

for app in "$APPDIR"/QGIS-final-*.app; do
  [[ -d "$app" ]] && quit_app_at_path "$app"
done

sudo rm -rf "$TMPDIR/$APP_NAME.bkp"
if [[ -d "$APPDIR/$APP_NAME" ]]; then
  sudo mv "$APPDIR/$APP_NAME" "$TMPDIR/$APP_NAME.bkp" || exit $?
fi
if ! sudo cp -R "$NEW_APP" "$APPDIR/"; then
  # remove the partial copy so a failed install isn't inventoried as the new
  # version, then restore the previous version if there was one
  sudo rm -rf "$APPDIR/$APP_NAME"
  if [[ -d "$TMPDIR/$APP_NAME.bkp" ]]; then
    sudo mv "$TMPDIR/$APP_NAME.bkp" "$APPDIR/$APP_NAME"
  fi
  exit 1
fi
sudo rm -rf "$TMPDIR/$APP_NAME.bkp"

for app in "$APPDIR"/QGIS-final-*.app; do
  if [[ -d "$app" && "$(basename "$app")" != "$APP_NAME" ]]; then
    echo "Removing earlier release $app"
    sudo rm -rf "$app"
  fi
done

echo "Installed $APPDIR/$APP_NAME"
