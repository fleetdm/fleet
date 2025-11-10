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

# Clean up any backup files that might exist from previous failed installations
# This ensures we start with a clean slate
cleanup_backup_files() {
  # Clean up backup in the installer's temp directory
  if [ -d "$TMPDIR/Microsoft Edge.app.bkp" ]; then
    echo "Removing existing backup file: $TMPDIR/Microsoft Edge.app.bkp"
    sudo rm -rf "$TMPDIR/Microsoft Edge.app.bkp" 2>/dev/null || true
  fi

  # Search for backup files in all common temp locations
  # Use -exec to avoid pipe subshell issues
  for search_base in /tmp /var/folders /private/var/folders; do
    if [ -d "$search_base" ]; then
      find "$search_base" -type d -name "Microsoft Edge.app.bkp" -exec sudo rm -rf {} + 2>/dev/null || true
    fi
  done
}

# copy to the applications folder
quit_application 'com.microsoft.edgemac'

# Clean up any existing backup files before creating a new one
cleanup_backup_files

# Remove existing app if present (like Homebrew does)
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	sudo rm -rf "$APPDIR/Microsoft Edge.app"
fi

# Install the new app
sudo cp -R "$TMPDIR/Microsoft Edge.app" "$APPDIR"

# Verify installation and do final cleanup
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	# Installation successful - ensure no backup files remain
	cleanup_backup_files
	echo "Installation verified"
else
	echo "Installation failed"
	exit 1
fi


