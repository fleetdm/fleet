#!/bin/bash

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
    if [[ "$quit_error" == *"(-600)" ]]; then
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

[ -n "$INSTALLER_PATH" ] && [ -f "$INSTALLER_PATH" ] || { echo "missing installer"; exit 1; }

quit_and_track_application "com.adobe.acc.AdobeCreativeCloud" || exit 1

# Mount to a known path to avoid space-in-volume issues
MOUNT_POINT="$(mktemp -d "/tmp/adobe_cc.XXXXXX")"
if ! hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null 2>&1; then
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "failed to mount dmg"
  exit 1
fi

# Find the installer app (handles Install.app or variants like *Install*.app)
INSTALL_APP="$(/usr/bin/find "$MOUNT_POINT" -maxdepth 5 -type d \( -name "Install.app" -o -iname "*Install*.app" \) -print -quit)"

if [ -z "$INSTALL_APP" ] || [ ! -d "$INSTALL_APP" ]; then
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "Install.app not found"
  exit 1
fi

# Prefer Contents/MacOS/Install, fall back to first executable in MacOS
BIN="$INSTALL_APP/Contents/MacOS/Install"
if [ ! -x "$BIN" ]; then
  BIN="$(/usr/bin/find "$INSTALL_APP/Contents/MacOS" -type f -perm +111 -print -quit 2>/dev/null)"
fi
[ -n "$BIN" ] && [ -x "$BIN" ] || { hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true; rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true; echo "installer binary not found"; exit 1; }

# Try silent, fall back to normal if needed
"$BIN" --mode=silent >/dev/null 2>&1 || "$BIN" >/dev/null 2>&1

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true

relaunch_application "com.adobe.acc.AdobeCreativeCloud"

echo "adobe creative cloud installed"
