#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(mktemp -d)

# functions

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
  return 1
}

app_pids() {
  local bundle_id="$1" uid out pid puid exe
  uid=$(id -u "$2") || return 1
  out=$(ps -axo pid=,uid=,comm=) || return 1
  while read -r pid puid exe; do
    [[ "$puid" == "$uid" && "$exe" == *.app/Contents/MacOS/* ]] || continue
    if grep -qixF -- "$bundle_id" <<< "$(defaults read "${exe%/Contents/MacOS/*}/Contents/Info" CFBundleIdentifier 2>/dev/null)"; then
      echo "$pid"
    fi
  done <<< "$out"
}

force_quit_application() {
  local bundle_id="$1" console_user="$2" pids
  if ! pids=$(app_pids "$bundle_id" "$console_user"); then
    echo "Couldn't check whether application '$bundle_id' is running."
    return 1
  fi
  if [[ -z "$pids" ]]; then
    echo "Found no process to stop for application '$bundle_id'."
    return 1
  fi
  echo "Stopping application '$bundle_id'..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    if pids=$(app_pids "$bundle_id" "$console_user") && [[ -z "$pids" ]]; then
      echo "Application '$bundle_id' stopped."
      return 0
    fi
  done
  if [[ -n "$pids" ]]; then
    echo "Application '$bundle_id' didn't stop; forcing it to quit."
    kill -KILL $pids 2>/dev/null
    for _ in {1..5}; do
      sleep 1
      if pids=$(app_pids "$bundle_id" "$console_user") && [[ -z "$pids" ]]; then
        echo "Application '$bundle_id' stopped."
        return 0
      fi
    done
  fi
  echo "Application '$bundle_id' did not stop."
  return 1
}

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
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
  local quit_success=false still_running quit_error
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    quit_error=$(osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" 2>&1 >/dev/null)
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
      echo "Application '$bundle_id' isn't running."
      quit_success=true
      break
    fi
    sleep 1
    if still_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
      echo "Application '$bundle_id' quit successfully."
      quit_success=true
      break
    fi
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
    [[ -z "$quit_error" ]] || echo "Last quit error: $quit_error"
    force_quit_application "$bundle_id" "$console_user" || return 1
  fi
}

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR" || exit 1

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

# vpn-installer.sh kills anything still running from the app before replacing it,
# so a failed quit doesn't stop the install.
quit_application 'com.express.vpn'
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

