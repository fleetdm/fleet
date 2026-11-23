#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

wezterm_pids() {
  local uid out pid puid exe
  uid=$(id -u "$1") || return 1
  out=$(ps -axo pid=,uid=,comm=) || return 1
  while read -r pid puid exe; do
    [[ "$puid" == "$uid" && "$exe" == */Contents/MacOS/wezterm-gui ]] || continue
    if [[ "$(defaults read "${exe%/Contents/MacOS/*}/Contents/Info" CFBundleIdentifier 2>/dev/null)" == "com.github.wez.wezterm" ]]; then
      echo "$pid"
    fi
  done <<< "$out"
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
  if ! pids=$(wezterm_pids "$console_user"); then
    echo "Couldn't check whether application '$bundle_id' is running."
    return 1
  fi
  [[ -n "$pids" ]] || return 0

  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

  echo "Quitting application '$bundle_id'..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    if pids=$(wezterm_pids "$console_user") && [[ -z "$pids" ]]; then
      echo "Application '$bundle_id' quit successfully."
      return 0
    fi
  done
  echo "Application '$bundle_id' did not quit."
  return 1
}


relaunch_application() {
  local bundle_id="$1"
  local app_path="${2:-}"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local was_running

  # Check if the app was running before installation. Default the variable so a
  # script running under 'set -u' doesn't abort when the quit helper never ran.
  eval "was_running=\${$var_name:-0}"
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
  local open_args=(-b "$bundle_id")
  [[ -z "$app_path" ]] || open_args=("$app_path")
  local timeout_duration=30
  local attempt=0 delay=2 open_status open_output
  SECONDS=0
  while true; do
    attempt=$((attempt + 1))
    open_status=0
    if [[ $EUID -eq 0 ]]; then
      local console_uid
      console_uid=$(id -u "$console_user")
      open_output=$(/bin/launchctl asuser "$console_uid" sudo -u "$console_user" open "${open_args[@]}" 2>&1) || open_status=$?
    else
      open_output=$(open "${open_args[@]}" 2>&1) || open_status=$?
    fi

    if [[ $open_status -eq 0 ]]; then
      echo "Application '$bundle_id' relaunched successfully."
      return
    fi
    echo "Attempt $attempt to relaunch application '$bundle_id' failed (exit $open_status): $open_output"
    # Back off 2, 4, 8, then 10 seconds. The budget bounds when the last
    # attempt may start, not how long one takes: a slow 'open' is usually
    # Gatekeeper checking the newly copied bundle, which is worth waiting out.
    (( SECONDS + delay <= timeout_duration )) || break
    sleep "$delay"
    # sleep only guarantees at least $delay, so a late wake-up — the Mac
    # suspending mid-sleep, say — can land past the budget. Check again rather
    # than trusting the estimate above; landing exactly on it still retries.
    (( SECONDS <= timeout_duration )) || break
    delay=$(( delay * 2 > 10 ? 10 : delay * 2 ))
  done
  echo "Failed to relaunch application '$bundle_id'."
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
relaunch_application 'com.github.wez.wezterm' "$APPDIR/WezTerm.app"
