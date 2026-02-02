#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    eval "export $var_name=0"
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    eval "export $var_name=0"
    return
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
    echo "Application '$bundle_id' did not quit."
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
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping relaunching application ID '$bundle_id'."
    return
  fi

  echo "Relaunching application '$bundle_id'..."

  # Try to launch the application
  if osascript -e "tell application id \"$bundle_id\" to activate" >/dev/null 2>&1; then
    echo "Application '$bundle_id' relaunched successfully."
  else
    echo "Failed to relaunch application '$bundle_id'."
  fi
}

# extract contents (zip from desktop.githubusercontent.com)
unzip "$INSTALLER_PATH" -d "$TMPDIR"

# Remove only the quarantine attribute. Do NOT use xattr -cr (clear all): that can
# invalidate the app's code signature and cause "damaged and can't be opened".
if [ -d "$TMPDIR/GitHub Desktop.app" ]; then
  xattr -dr com.apple.quarantine "$TMPDIR/GitHub Desktop.app" 2>/dev/null || true
fi

# copy to the applications folder
quit_and_track_application 'com.github.GitHubClient'
if [ -d "$APPDIR/GitHub Desktop.app" ]; then
  sudo mv "$APPDIR/GitHub Desktop.app" "$TMPDIR/GitHub Desktop.app.bkp"
fi
sudo cp -R "$TMPDIR/GitHub Desktop.app" "$APPDIR"

# Clear quarantine on the installed app (in case copy ran as root and preserved it).
sudo xattr -dr com.apple.quarantine "$APPDIR/GitHub Desktop.app" 2>/dev/null || true

relaunch_application 'com.github.GitHubClient'

mkdir -p .
/bin/ln -h -f -s -- "$APPDIR/GitHub Desktop.app/Contents/Resources/app/static/github.sh" "github"
