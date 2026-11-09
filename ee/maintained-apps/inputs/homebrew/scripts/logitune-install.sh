#!/bin/bash

quit_application() {
  local bundle_id="$1"
  local console_user="$2"
  local timeout_duration=10

  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

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

CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

# Quit Logi Tune gracefully before the PKG's preinstall force-kills it, so a
# failed quit doesn't stop the install. The PKG's default RUNAPP choice
# relaunches the app for logged-in console users after installation, so no
# relaunch step is needed here.
LOGITUNE_RUNNING=$(osascript -e "application id \"com.logitech.logitune\" is running" 2>/dev/null)
if [[ "$LOGITUNE_RUNNING" == "true" ]]; then
  quit_application 'com.logitech.logitune' "$CONSOLE_USER"
fi

installer -pkg "$INSTALLER_PATH" -target /
