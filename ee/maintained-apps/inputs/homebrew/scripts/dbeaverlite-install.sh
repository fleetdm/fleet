#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# functions

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

# extract contents
# Use 'yes' to automatically accept license agreement when mounting DMG
# This replicates Homebrew's behavior for DMG files with license agreements
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT" || true

# copy to the applications folder
quit_application 'com.dbeaver.product.lite'
if [ -d "$APPDIR/DBeaverLite.app" ]; then
	sudo mv "$APPDIR/DBeaverLite.app" "$TMPDIR/DBeaverLite.app.bkp"
fi
sudo cp -R "$TMPDIR/DBeaverLite.app" "$APPDIR"
