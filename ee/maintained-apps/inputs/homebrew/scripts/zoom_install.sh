#!/bin/bash

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
  return 1
}

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
  local quit_success=false quit_sent=false still_running quit_error
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    quit_error=$(osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" 2>&1 >/dev/null)
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
      if [[ "$quit_sent" == false ]]; then
        ZOOM_WAS_RUNNING=false
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

restart_zoom() {
  local console_user="$1"
  
  if [[ -n "$console_user" && "$console_user" != "root" ]]; then
    echo "Restarting Zoom for user: $console_user"
    sudo -u "$console_user" open -a "zoom.us"
  else
    echo "No console user found, attempting direct Zoom start..."
    open -a "zoom.us"
  fi
}

# Get console user once (used by both quit and restart)
CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

# Check if Zoom is running
ZOOM_WAS_RUNNING=false
ZOOM_RUNNING=$(osascript -e "application id \"us.zoom.xos\" is running" 2>/dev/null)
if [[ "$ZOOM_RUNNING" == "true" ]]; then
  ZOOM_WAS_RUNNING=true
  quit_application 'us.zoom.xos' "$CONSOLE_USER" || exit 1
fi

installer -pkg "$INSTALLER_PATH" -target / || exit $?

# Restart Zoom if it was running before installation
if [[ "$ZOOM_WAS_RUNNING" == "true" ]]; then
  sleep 2
  restart_zoom "$CONSOLE_USER" || true
fi

