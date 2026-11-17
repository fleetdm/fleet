#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
  return 1
}


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
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
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
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
# The disk image's RoboForm.app is an installer that unpacks the real app from
# a zip in its resources, so install the real app from that zip.
APP_ZIP=$(compgen -G "$MOUNT_POINT/RoboForm.app/Contents/Resources/roboform-mac-*.zip" | head -n 1)
if [[ -z "$APP_ZIP" ]] || ! sudo ditto -x -k "$APP_ZIP" "$TMPDIR"; then
	echo "Couldn't extract RoboForm.app from the disk image."
	hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
	exit 1
fi
hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
# copy to the applications folder
quit_and_track_application 'com.sibersystems.RoboFormMac' || exit 1
# RoboForm 9 reports com.SiberSystems.RoboForm from the same path.
if ! quit_and_track_application 'com.SiberSystems.RoboForm'; then
	relaunch_application 'com.sibersystems.RoboFormMac'
	exit 1
fi
if [ -d "$APPDIR/RoboForm.app" ]; then
	sudo mv "$APPDIR/RoboForm.app" "$TMPDIR/RoboForm.app.bkp" || exit $?
fi
if ! sudo cp -R "$TMPDIR/RoboForm.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/RoboForm.app"
	if [ -d "$TMPDIR/RoboForm.app.bkp" ]; then
		sudo mv "$TMPDIR/RoboForm.app.bkp" "$APPDIR/RoboForm.app"
	fi
	exit 1
fi
if [[ "$APP_WAS_RUNNING_com_SiberSystems_RoboForm" == "1" ]]; then
	# Register the new bundle ID with the user's Launch Services so it can be reopened.
	console_user=$(stat -f "%Su" /dev/console)
	/bin/launchctl asuser "$(id -u "$console_user")" sudo -u "$console_user" \
		/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f /Applications/RoboForm.app >/dev/null 2>&1 || true
	export APP_WAS_RUNNING_com_sibersystems_RoboFormMac=1
fi
relaunch_application 'com.sibersystems.RoboFormMac'
