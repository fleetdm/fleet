#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}

[[ -n "$INSTALLER_PATH" && -f "$INSTALLER_PATH" ]] || { echo "missing installer"; exit 1; }

APPDIR="/Applications"

quit_application "com.evernote.Evernote"

MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\//{print $3; exit}')"
[[ -n "$MOUNT_POINT" ]] || { echo "failed to mount dmg"; exit 1; }

if [[ -d "$MOUNT_POINT/Evernote.app" ]]; then
  rm -rf "$APPDIR/Evernote.app" >/dev/null 2>&1 || true
  ditto "$MOUNT_POINT/Evernote.app" "$APPDIR/Evernote.app" >/dev/null 2>&1
fi

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true

echo "Evernote installed"

