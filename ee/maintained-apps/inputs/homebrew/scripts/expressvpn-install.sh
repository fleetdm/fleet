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
unzip "$INSTALLER_PATH" -d "$TMPDIR"

# find the pkg file in the extracted directory (search recursively)
PKG_FILE=$(find "$TMPDIR" -name "*.pkg" -type f | head -n 1)

# install the pkg if found
if [ -n "$PKG_FILE" ] && [ -f "$PKG_FILE" ]; then
  quit_application 'com.expressvpn.ExpressVPN'
  sudo installer -pkg "$PKG_FILE" -target /
  EXIT_CODE=$?
  if [ $EXIT_CODE -ne 0 ]; then
    echo "Error: Installer exited with code $EXIT_CODE"
    exit $EXIT_CODE
  fi
else
  echo "Error: No pkg file found in $TMPDIR"
  exit 1
fi

