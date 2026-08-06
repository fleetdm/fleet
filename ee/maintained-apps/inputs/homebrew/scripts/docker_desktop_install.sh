#!/bin/bash

set -euo pipefail

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
MOUNT_POINT=""

cleanup() {
  local mp="${MOUNT_POINT:-}"
  if [[ -n "$mp" ]]; then
    if mount | grep -q " on $mp "; then
      hdiutil detach "$mp" >/dev/null 2>&1 || true
    fi
    rmdir "$mp" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name
  var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null || echo "false")
  if [[ "$app_running" != "true" ]]; then
    eval "export $var_name=0"
    return 0
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    eval "export $var_name=0"
    return 0
  fi

  # App was running, mark it for relaunch
  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

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
    echo "Application '$bundle_id' did not quit within ${timeout_duration}s; aborting install." >&2
    return 1
  fi
}


relaunch_application() {
  local bundle_id="$1"
  local var_name
  var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local was_running

  # Check if the app was running before installation
  eval "was_running=\${$var_name:-0}"
  if [[ "$was_running" != "1" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")
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
if ! hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"; then
  echo "Failed to mount DMG '$INSTALLER_PATH'." >&2
  exit 1
fi
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"
MOUNT_POINT=""
# copy to the applications folder
# Quitting Docker Desktop with a staged self-update triggers its install-on-quit
# updater, which renames Docker.app to Docker.app.back and races this script.
# Remove the staging dir (staged bundle + updater state) first so it can't fire.
sudo rm -rf /Users/*/Library/"Application Support"/com.docker.install
quit_and_track_application 'com.electron.dockerdesktop'
# Wait out any updater already in flight before touching /Applications/Docker.app.
SECONDS=0
while pgrep -f 'com\.docker\.install' >/dev/null 2>&1 && (( SECONDS < 30 )); do
  sleep 1
done
if [ -d "$APPDIR/Docker.app" ]; then
	sudo mv "$APPDIR/Docker.app" "$TMPDIR/Docker.app.bkp"
fi
# Remove stale self-updater leftovers; osquery's apps table picks them up by
# bundle_identifier and patch policies report Docker as out of date.
sudo rm -rf "$APPDIR/Docker.app.back"
sudo cp -R "$TMPDIR/Docker.app" "$APPDIR"
relaunch_application 'com.electron.dockerdesktop'
mkdir -p /usr/local/cli-plugins
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/cli-plugins/docker-compose" "/usr/local/cli-plugins/docker-compose"
mkdir -p /usr/local/bin
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/hub-tool" "/usr/local/bin/hub-tool"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/kubectl" "/usr/local/bin/kubectl.docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker" "/usr/local/bin/docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-desktop" "/usr/local/bin/docker-credential-desktop"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-ecr-login" "/usr/local/bin/docker-credential-ecr-login"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain" "/usr/local/bin/docker-credential-osxkeychain"
# Remove stale copies recreated during the quit/relaunch window, if any.
sudo rm -rf "$APPDIR/Docker.app.back"
sudo rm -rf /Users/*/Library/"Application Support"/com.docker.install/in_progress/Docker.app
