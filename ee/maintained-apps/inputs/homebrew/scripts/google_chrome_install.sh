#!/bin/sh

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

restart_chrome() {
  local console_user="$1"
  
  if [[ -n "$console_user" && "$console_user" != "root" ]]; then
    echo "Restarting Chrome for user: $console_user"
    sudo -u "$console_user" open -a "Google Chrome" --args --restore-last-session
  else
    echo "No console user found, attempting direct Chrome start..."
    open -a "Google Chrome" --args --restore-last-session
  fi
}

# Get console user once (used by both quit and restart)
CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

# Check if Chrome is running (only check once)
CHROME_WAS_RUNNING=false
if osascript -e "application id \"com.google.Chrome\" is running" 2>/dev/null; then
  CHROME_WAS_RUNNING=true
  quit_application 'com.google.Chrome' "$CONSOLE_USER"
fi

installer -pkg "$INSTALLER_PATH" -target /

# Restart Chrome if it was running before installation
if [[ "$CHROME_WAS_RUNNING" == "true" ]]; then
  sleep 2
  restart_chrome "$CONSOLE_USER" || true
fi