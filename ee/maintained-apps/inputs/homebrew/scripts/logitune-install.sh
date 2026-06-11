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

CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

# Quit Logi Tune gracefully before the PKG's preinstall force-kills it. The
# PKG's default RUNAPP choice relaunches the app for logged-in console users
# after installation, so no relaunch step is needed here.
if osascript -e "application id \"com.logitech.logitune\" is running" 2>/dev/null; then
  quit_application 'com.logitech.logitune' "$CONSOLE_USER"
fi

installer -pkg "$INSTALLER_PATH" -target /
