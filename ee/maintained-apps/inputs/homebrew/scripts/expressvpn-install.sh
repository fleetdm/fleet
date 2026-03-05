#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(mktemp -d)

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

# discover the installer app by finding any .app that contains an installer executable
INSTALLER_APP=""
for app in "$TMPDIR"/*.app; do
  if [ -d "$app" ] && [ -d "$app/Contents/MacOS" ]; then
    INSTALLER_APP="$app"
    break
  fi
done

if [ -z "$INSTALLER_APP" ] || [ ! -d "$INSTALLER_APP" ]; then
  echo "Error: Installer app not found in $TMPDIR"
  exit 1
fi

quit_application 'com.expressvpn.ExpressVPN'

# Remove quarantine attributes so Gatekeeper won't block binaries during install
sudo xattr -r -d com.apple.quarantine "$INSTALLER_APP" 2>/dev/null || true

# Run the bundled installer script which handles copying to /Applications,
# setting permissions, creating groups, installing the LaunchDaemon, and
# starting the daemon
INSTALLER_SCRIPT="$INSTALLER_APP/Contents/Resources/vpn-installer.sh"
if [ ! -f "$INSTALLER_SCRIPT" ]; then
  echo "Error: vpn-installer.sh not found in $INSTALLER_APP/Contents/Resources"
  exit 1
fi

chmod +x "$INSTALLER_SCRIPT"
sudo bash "$INSTALLER_SCRIPT"
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
  echo "Error: Installer exited with code $EXIT_CODE"
  exit $EXIT_CODE
fi

