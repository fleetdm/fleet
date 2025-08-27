#!/bin/sh
set -e
APPDIR="/Applications"
TMPDIR="$(dirname "$(realpath "$INSTALLER_PATH")")"

quit_app() {
  b="$1"
  if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then
    cu="$(stat -f "%Su" /dev/console || true)"
    [ "$cu" = "root" ] && return 0
    osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true
    sleep 2
  fi
}

MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\// {print $3; exit}')"
[ -n "$MOUNT_POINT" ] || { echo "failed to mount dmg"; exit 1; }

cleanup() {
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for app in p4v.app p4merge.app p4admin.app; do
  [ -d "$MOUNT_POINT/$app" ] || continue
  [ "$app" = "p4v.app" ] && quit_app "com.perforce.p4v"
  [ -d "$APPDIR/$app" ] && sudo mv "$APPDIR/$app" "$TMPDIR/$app.bkp" || true
  sudo cp -R "$MOUNT_POINT/$app" "$APPDIR/"
done
echo "p4v installed"
