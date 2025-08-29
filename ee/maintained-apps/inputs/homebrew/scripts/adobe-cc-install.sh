#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Skipping quit for '$bundle_id'."
    return
  fi

  echo "Quitting '$bundle_id'..."
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1 || true
    if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
      echo "'$bundle_id' quit successfully."
      return
    fi
    sleep 1
  done
  echo "'$bundle_id' did not quit."
}

quit_application "com.adobe.acc.AdobeCreativeCloud"

MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\// {print $3; exit}')"
[ -n "$MOUNT_POINT" ] || { echo "failed to mount dmg"; exit 1; }
trap "hdiutil detach \"$MOUNT_POINT\" >/dev/null 2>&1 || true" EXIT

BIN="$(/usr/bin/find "$MOUNT_POINT" -type f -path '*/Install.app/Contents/MacOS/Install' -print -quit)"
[ -z "$BIN" ] && BIN="$(/usr/bin/find "$MOUNT_POINT" -type f -path '*/Creative Cloud Installer.app/Contents/MacOS/Install' -print -quit)"
[ -n "$BIN" ] || { echo "installer binary not found"; exit 1; }

sudo "$BIN" --mode=silent
echo "adobe creative cloud installed"
