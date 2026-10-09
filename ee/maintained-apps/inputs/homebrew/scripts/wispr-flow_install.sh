#!/bin/bash

# Wispr Flow auto-updates in place as the logged-in user via Squirrel.Mac. When
# the app bundle is owned by root — the default for an install that runs as
# root — those updates fail and Wispr Flow repeatedly prompts the user for
# admin access to fix the bundle's ownership, so this script assigns
# ownership to the console user.

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
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
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1
    sleep 1
    if still_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
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
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
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
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT" || true
# copy to the applications folder
quit_and_track_application 'com.electron.wispr-flow' || exit 1
if [ -d "$APPDIR/Wispr Flow.app" ]; then
	sudo mv "$APPDIR/Wispr Flow.app" "$TMPDIR/Wispr Flow.app.bkp" || exit $?
fi
if ! sudo cp -R "$TMPDIR/Wispr Flow.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/Wispr Flow.app"
	if [ -d "$TMPDIR/Wispr Flow.app.bkp" ]; then
		sudo mv "$TMPDIR/Wispr Flow.app.bkp" "$APPDIR/Wispr Flow.app"
	fi
	exit 1
fi

target_user=$(stat -f "%Su" /dev/console)
if [[ -z "$target_user" || "$target_user" == "root" || "$target_user" == "loginwindow" || "$target_user" == "_mbsetupuser" ]]; then
  # No GUI session (e.g. install triggered while logged out): fall back to the
  # last user that logged in.
  target_user=$(defaults read /Library/Preferences/com.apple.loginwindow lastUserName 2>/dev/null)
fi
if [[ -n "$target_user" && "$target_user" != "root" && "$target_user" != "_mbsetupuser" ]] && id -u "$target_user" >/dev/null 2>&1; then
  if sudo chown -R "$target_user":staff "$APPDIR/Wispr Flow.app" && sudo chmod -R u+w "$APPDIR/Wispr Flow.app"; then
    echo "Assigned ownership of Wispr Flow.app to '$target_user' so Wispr Flow can auto-update."
  else
    echo "Failed to assign ownership of Wispr Flow.app to '$target_user'; Wispr Flow will prompt the user to fix ownership before it can auto-update."
  fi
else
  echo "No logged-in (or last logged-in) user found; Wispr Flow.app stays owned by root and Wispr Flow will prompt the user to fix ownership before it can auto-update."
fi

relaunch_application 'com.electron.wispr-flow'
