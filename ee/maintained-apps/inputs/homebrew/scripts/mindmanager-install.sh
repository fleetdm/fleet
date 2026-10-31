#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local timeout_duration=10

  # check if the application is running
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

  # App was running, mark it for relaunch
  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false still_running
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" >/dev/null 2>&1
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
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
if ! sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"; then
	hdiutil detach "$MOUNT_POINT" || true
	exit 1
fi
hdiutil detach "$MOUNT_POINT" || true
# MindManager's bundle ID changes with each major version, so read the new one
# from the installer, quit an older major that's still installed, and relaunch
# the new version in its place.
NEW_BUNDLE_ID=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$TMPDIR/MindManager.app/Contents/Info.plist" 2>/dev/null)
if [[ ! "$NEW_BUNDLE_ID" =~ ^com\.mindjet\.mindmanager\.[0-9]+$ ]]; then
	echo "Unexpected MindManager bundle ID '$NEW_BUNDLE_ID'."
	exit 1
fi
# copy to the applications folder
quit_and_track_application "$NEW_BUNDLE_ID" || exit 1
OLD_BUNDLE_ID=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APPDIR/MindManager.app/Contents/Info.plist" 2>/dev/null)
if [[ ! "$OLD_BUNDLE_ID" =~ ^com\.mindjet\.mindmanager\.[0-9]+$ ]]; then
	OLD_BUNDLE_ID=""
fi
if [[ -n "$OLD_BUNDLE_ID" && "$OLD_BUNDLE_ID" != "$NEW_BUNDLE_ID" ]]; then
	if ! quit_and_track_application "$OLD_BUNDLE_ID"; then
		# reopen the new version if it was running from elsewhere and quit above
		relaunch_application "$NEW_BUNDLE_ID"
		exit 1
	fi
	OLD_WAS_RUNNING_VAR="APP_WAS_RUNNING_${OLD_BUNDLE_ID//[^[:alnum:]_]/_}"
	if [[ "${!OLD_WAS_RUNNING_VAR}" == "1" ]]; then
		export "APP_WAS_RUNNING_${NEW_BUNDLE_ID//[^[:alnum:]_]/_}=1"
	fi
fi
if [ -d "$APPDIR/MindManager.app" ]; then
	sudo mv "$APPDIR/MindManager.app" "$TMPDIR/MindManager.app.bkp" || exit $?
fi
if ! sudo cp -R "$TMPDIR/MindManager.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one and reopen it
	# if it was quit for the update
	sudo rm -rf "$APPDIR/MindManager.app"
	if [ -d "$TMPDIR/MindManager.app.bkp" ] && sudo mv "$TMPDIR/MindManager.app.bkp" "$APPDIR/MindManager.app" && [[ -n "$OLD_BUNDLE_ID" ]]; then
		relaunch_application "$OLD_BUNDLE_ID"
	fi
	exit 1
fi
relaunch_application "$NEW_BUNDLE_ID"
