#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
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
  local bundle_id="$1" console_user="$2" pids kill_cmd=(kill)
  [[ $EUID -ne 0 ]] || kill_cmd=(sudo -u "$console_user" kill)
  if ! pids=$(app_pids "$bundle_id" "$console_user"); then
    echo "Couldn't check whether application '$bundle_id' is running."
    return 1
  fi
  if [[ -z "$pids" ]]; then
    if ! app_process_running "$bundle_id" &&
      [[ "$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)" == "false" ]]; then
      echo "Application '$bundle_id' is no longer running."
      return 0
    fi
    echo "Found no process to stop for application '$bundle_id'."
    return 1
  fi
  echo "Stopping application '$bundle_id'..."
  "${kill_cmd[@]}" -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    if pids=$(app_pids "$bundle_id" "$console_user") && [[ -z "$pids" ]]; then
      echo "Application '$bundle_id' stopped."
      return 0
    fi
  done
  if [[ -n "$pids" ]]; then
    echo "Application '$bundle_id' didn't stop; forcing it to quit."
    "${kill_cmd[@]}" -KILL $pids 2>/dev/null
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
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
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
    [[ -z "$quit_error" ]] || echo "Last quit error: $quit_error"
    force_quit_application "$bundle_id" "$console_user" || return 1
  fi
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
    # Back off 2, 4, 8, then 10 seconds, stopping before the budget is spent
    # rather than after, so a failing relaunch never runs past it.
    (( SECONDS + delay <= timeout_duration )) || break
    sleep "$delay"
    delay=$(( delay * 2 > 10 ? 10 : delay * 2 ))
  done
  echo "Failed to relaunch application '$bundle_id'."
}


# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
if ! sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"; then
	hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
	exit 1
fi
hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
# MindManager's bundle ID changes with each major version, so read the new one
# from the installer, quit an older major that's still installed, and relaunch
# the new version in its place.
NEW_BUNDLE_ID=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$TMPDIR/MindManager.app/Contents/Info.plist" 2>/dev/null)
if [[ ! "$NEW_BUNDLE_ID" =~ ^com\.mindjet\.mindmanager\.[0-9]+$ ]]; then
	echo "Unexpected MindManager bundle ID '$NEW_BUNDLE_ID'."
	exit 1
fi
# copy to the applications folder
quit_and_track_application "$NEW_BUNDLE_ID" || exit 1
NEW_WAS_RUNNING_VAR="APP_WAS_RUNNING_${NEW_BUNDLE_ID//[^[:alnum:]_]/_}"
NEW_WAS_RUNNING="${!NEW_WAS_RUNNING_VAR}"
OLD_BUNDLE_ID=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APPDIR/MindManager.app/Contents/Info.plist" 2>/dev/null)
if [[ ! "$OLD_BUNDLE_ID" =~ ^com\.mindjet\.mindmanager\.[0-9]+$ ]]; then
	OLD_BUNDLE_ID=""
fi
if [[ -n "$OLD_BUNDLE_ID" && "$OLD_BUNDLE_ID" != "$NEW_BUNDLE_ID" ]]; then
	if ! quit_and_track_application "$OLD_BUNDLE_ID"; then
		# reopen the new version if it was running from elsewhere and quit above
		relaunch_application "$NEW_BUNDLE_ID"
		exit 1
	fi
	OLD_WAS_RUNNING_VAR="APP_WAS_RUNNING_${OLD_BUNDLE_ID//[^[:alnum:]_]/_}"
	if [[ "${!OLD_WAS_RUNNING_VAR}" == "1" ]]; then
		export "$NEW_WAS_RUNNING_VAR=1"
	fi
fi
# After a failed install, reopen what was quit for it: a new-major copy running
# from elsewhere, and the previous version if it's in place again.
reopen_quit_apps() {
	local previous_in_place="$1"
	export "$NEW_WAS_RUNNING_VAR=$NEW_WAS_RUNNING"
	if [[ "$OLD_BUNDLE_ID" != "$NEW_BUNDLE_ID" || "$previous_in_place" == "1" ]]; then
		relaunch_application "$NEW_BUNDLE_ID"
	fi
	if [[ -n "$OLD_BUNDLE_ID" && "$OLD_BUNDLE_ID" != "$NEW_BUNDLE_ID" && "$previous_in_place" == "1" ]]; then
		relaunch_application "$OLD_BUNDLE_ID"
	fi
}
if [ -d "$APPDIR/MindManager.app" ]; then
	if ! sudo mv "$APPDIR/MindManager.app" "$TMPDIR/MindManager.app.bkp"; then
		reopen_quit_apps 1
		exit 1
	fi
fi
if ! sudo cp -R "$TMPDIR/MindManager.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/MindManager.app"
	restored=0
	if [ -d "$TMPDIR/MindManager.app.bkp" ] && sudo mv "$TMPDIR/MindManager.app.bkp" "$APPDIR/MindManager.app"; then
		restored=1
	fi
	reopen_quit_apps "$restored"
	exit 1
fi
relaunch_application "$NEW_BUNDLE_ID" "$APPDIR/MindManager.app"
