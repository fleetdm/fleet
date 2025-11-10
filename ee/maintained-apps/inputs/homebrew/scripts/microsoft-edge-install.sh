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
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"

# copy to the applications folder
quit_application 'com.microsoft.edgemac'
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	sudo mv "$APPDIR/Microsoft Edge.app" "$TMPDIR/Microsoft Edge.app.bkp"
fi
sudo cp -R "$TMPDIR/Microsoft Edge.app" "$APPDIR"

# Clean up backup file if installation was successful
# Always attempt cleanup, even if copy might have failed
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	if [ -d "$TMPDIR/Microsoft Edge.app.bkp" ]; then
		sudo rm -rf "$TMPDIR/Microsoft Edge.app.bkp"
		echo "Installation verified, backup file removed"
	fi
else
	# If installation failed, restore the backup
	if [ -d "$TMPDIR/Microsoft Edge.app.bkp" ]; then
		echo "Installation failed, restoring backup"
		sudo mv "$TMPDIR/Microsoft Edge.app.bkp" "$APPDIR/Microsoft Edge.app"
	fi
	exit 1
fi


