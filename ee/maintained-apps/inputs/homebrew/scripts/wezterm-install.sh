#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

wezterm_pids() {
  ps -x -U "$1" -o pid=,comm= | sed -n 's|^ *\([0-9][0-9]*\) .*/WezTerm\.app/Contents/MacOS/wezterm-gui$|\1|p'
}

# By default WezTerm asks "Terminate WezTerm?" before it quits, so the
# AppleScript quit never finishes. SIGTERM closes it without asking.
quit_and_track_wezterm() {
  local bundle_id="com.github.wez.wezterm"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  eval "export $var_name=0"

  local console_user pids
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi
  pids=$(wezterm_pids "$console_user")
  [[ -n "$pids" ]] || return 0

  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

  echo "Quitting application '$bundle_id'..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    if [[ -z "$(wezterm_pids "$console_user")" ]]; then
      echo "Application '$bundle_id' quit successfully."
      return 0
    fi
  done
  echo "Application '$bundle_id' did not quit."
  return 1
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


# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR" || exit $?
# the zip nests WezTerm.app inside a versioned folder
SRC=$(find "$TMPDIR" -maxdepth 2 -name WezTerm.app -type d | head -n 1)
if [ -z "$SRC" ]; then
	echo "WezTerm.app not found in the installer."
	exit 1
fi
quit_and_track_wezterm || exit 1
if [ -d "$APPDIR/WezTerm.app" ]; then
	sudo mv "$APPDIR/WezTerm.app" "$TMPDIR/WezTerm.app.bkp" || exit $?
fi
if ! sudo cp -R "$SRC" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/WezTerm.app"
	if [ -d "$TMPDIR/WezTerm.app.bkp" ]; then
		sudo mv "$TMPDIR/WezTerm.app.bkp" "$APPDIR/WezTerm.app"
	fi
	exit 1
fi
relaunch_application 'com.github.wez.wezterm'
