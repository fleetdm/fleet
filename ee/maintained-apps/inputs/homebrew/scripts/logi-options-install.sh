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

quit_application "com.logi.optionsplus"

TMPDIR="$(mktemp -d)"
unzip -q "$INSTALLER_PATH" -d "$TMPDIR"

APP="$(/usr/bin/find "$TMPDIR" -maxdepth 3 -type d -name "logioptionsplus_installer.app" -print -quit)"
[[ -d "$APP" ]] || { echo "installer app not found"; rm -rf "$TMPDIR"; exit 1; }

"$APP/Contents/MacOS/logioptionsplus_installer" --quiet >/dev/null 2>&1 || true

rm -rf "$TMPDIR" >/dev/null 2>&1 || true

echo "logi options+ installed"
