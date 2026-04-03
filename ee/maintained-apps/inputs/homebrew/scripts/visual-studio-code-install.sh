#!/bin/sh

APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

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

# Get console user once
CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

# Check if VS Code has visible windows (not just background helpers like Code Helper, Code Helper (Renderer), etc.)
VSCODE_WAS_RUNNING=false
if osascript -e "application id \"com.microsoft.VSCode\" is running" 2>/dev/null; then
  # VS Code registers as "running" even when only background helper processes are active.
  # Verify it has visible windows before treating it as running.
  window_count=$(osascript -e 'tell application "System Events"
    set appRunning to (bundle identifier of processes) contains "com.microsoft.VSCode"
    if not appRunning then return 0
    return count of windows of (first process whose bundle identifier is "com.microsoft.VSCode")
  end tell' 2>/dev/null || echo "0")
  if [[ "$window_count" -gt 0 ]]; then
    VSCODE_WAS_RUNNING=true
    quit_application 'com.microsoft.VSCode' "$CONSOLE_USER"
  else
    echo "VS Code has no visible windows; skipping quit and relaunch."
  fi
fi

# Extract and install
unzip "$INSTALLER_PATH" -d "$TMPDIR"
if [ -d "$APPDIR/Visual Studio Code.app" ]; then
  sudo mv "$APPDIR/Visual Studio Code.app" "$TMPDIR/Visual Studio Code.app.bkp"
fi
sudo cp -R "$TMPDIR/Visual Studio Code.app" "$APPDIR"

# Relaunch only if it had visible windows
if [[ "$VSCODE_WAS_RUNNING" == "true" ]]; then
  if [[ -n "$CONSOLE_USER" && "$CONSOLE_USER" != "root" ]]; then
    echo "Relaunching VS Code for user: $CONSOLE_USER"
    sudo -u "$CONSOLE_USER" open -a "Visual Studio Code"
  else
    open -a "Visual Studio Code"
  fi
fi
