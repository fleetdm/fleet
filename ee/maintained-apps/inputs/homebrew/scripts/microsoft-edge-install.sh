#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")

# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local timeout_duration=10

  local app_running
  app_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    eval "export $var_name=0"
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    eval "export $var_name=0"
    return
  fi

  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

  echo "Quitting application '$bundle_id'..."

  local quit_success=false quit_sent=false still_running quit_error
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    quit_error=$(osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" 2>&1 >/dev/null)
    if [[ "$quit_error" == *"(-600)" ]]; then
      if [[ "$quit_sent" == false ]]; then
        eval "export $var_name=0"
        echo "Application '$bundle_id' isn't running; it won't be relaunched."
      else
        echo "Application '$bundle_id' quit successfully."
      fi
      quit_success=true
      break
    fi
    quit_sent=true
    sleep 1
    if still_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
      echo "Application '$bundle_id' quit successfully."
      quit_success=true
      break
    fi
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
    return 1
  fi
}

relaunch_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local was_running

  # Check if the app was running before installation
  eval "was_running=\$$var_name"
  if [[ "$was_running" != "1" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping relaunching application ID '$bundle_id'."
    return
  fi

  echo "Relaunching application '$bundle_id'..."

  # Launch the app in the logged-in user's GUI session. Apps launched by root
  # won't register with the user's Dock/GUI, so run 'open' as the console user.
  # Use 'launchctl asuser' to bootstrap into the console user's Mach namespace
  # and GUI session — 'sudo -u' alone doesn't do this, which can cause
  # LSOpenURLsWithRole() failures even when 'open' exits 0.
  local open_status=0
  if [[ $EUID -eq 0 ]]; then
    local console_uid
    console_uid=$(id -u "$console_user")
    /bin/launchctl asuser "$console_uid" sudo -u "$console_user" open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  else
    open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  fi

  if [[ $open_status -eq 0 ]]; then
    echo "Application '$bundle_id' relaunched successfully."
  else
    echo "Failed to relaunch application '$bundle_id'."
  fi
}

# extract contents
# Fail before the existing app is removed below, so a bad download can't leave
# the host without a working install.
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
if ! hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"; then
	echo "Failed to mount DMG '$INSTALLER_PATH'." >&2
	exit 1
fi
if ! sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"; then
	hdiutil detach "$MOUNT_POINT" || true
	exit 1
fi
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

remove_stale_updater_bundles() {
  local root
  for root in / /Users/*; do
    local updater_dir="${root%/}/Library/Application Support/Microsoft/EdgeUpdater"
    [ -d "$updater_dir" ] || continue
    find "$updater_dir" -type d -name "Microsoft Edge.app" -prune -exec sudo rm -rf {} + 2>/dev/null || true
  done
}

# copy to the applications folder
quit_and_track_application 'com.microsoft.edgemac' || exit 1

# Clean up any existing backup files before creating a new one
cleanup_backup_files

# Remove existing app if present (like Homebrew does)
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	sudo rm -rf "$APPDIR/Microsoft Edge.app"
fi

# Install the new app
if ! sudo cp -R "$TMPDIR/Microsoft Edge.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new version
	sudo rm -rf "$APPDIR/Microsoft Edge.app"
	echo "Installation failed"
	exit 1
fi

# Verify installation and do final cleanup
if [ -d "$APPDIR/Microsoft Edge.app" ]; then
	# Installation successful - ensure no backup files remain
	cleanup_backup_files
	remove_stale_updater_bundles
	echo "Installation verified"
else
	echo "Installation failed"
	exit 1
fi

relaunch_application 'com.microsoft.edgemac'
