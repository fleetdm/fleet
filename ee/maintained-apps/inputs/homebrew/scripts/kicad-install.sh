#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
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
    eval "export $var_name=0"
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


detach_dmg() {
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || hdiutil detach -force "$MOUNT_POINT" >/dev/null 2>&1 || true
}

# detach the disk image, relaunch any app that was quit, and fail the install
abort_install() {
  detach_dmg
  local bundle_id
  for bundle_id in "${BUNDLE_IDS[@]}"; do
    relaunch_application "$bundle_id"
  done
  exit 1
}

# KiCad installs as a folder of apps (KiCad, Schematic Editor, PCB Editor, and
# others) in /Applications/KiCad. Every app in that folder is quit before the
# folder is replaced, and the install stops if one won't quit (for example, the
# user cancels a prompt to save changes). The demos folder and command-line
# tool links that the disk image also offers aren't installed.
installed_bundle_ids() {
  local app
  for app in "$APPDIR/KiCad"/*.app; do
    [[ -d "$app" ]] || continue
    /usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$app/Contents/Info.plist" 2>/dev/null | grep -E '^[A-Za-z0-9._-]+$'
  done
}

# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null || exit 1
if [ ! -d "$MOUNT_POINT/KiCad" ]; then
	echo "KiCad folder not found in the disk image."
	detach_dmg
	exit 1
fi

BUNDLE_IDS=()
while IFS= read -r bundle_id; do
	[[ -n "$bundle_id" ]] && BUNDLE_IDS+=("$bundle_id")
done < <(installed_bundle_ids)
for bundle_id in "${BUNDLE_IDS[@]}"; do
	quit_and_track_application "$bundle_id" || abort_install
done

# copy to the applications folder
if [ -d "$APPDIR/KiCad" ]; then
	sudo mv "$APPDIR/KiCad" "$TMPDIR/KiCad.bkp" || abort_install
fi
if ! sudo cp -R "$MOUNT_POINT/KiCad" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/KiCad"
	if [ -d "$TMPDIR/KiCad.bkp" ]; then
		sudo mv "$TMPDIR/KiCad.bkp" "$APPDIR/KiCad"
	fi
	abort_install
fi
detach_dmg

# Apps in a subfolder of /Applications are only inventoried once LaunchServices
# has registered them.
lsregister_status=0
"$LSREGISTER" -f "$APPDIR/KiCad/KiCad.app" || lsregister_status=$?

for bundle_id in "${BUNDLE_IDS[@]}"; do
	relaunch_application "$bundle_id"
done
exit $lsregister_status
