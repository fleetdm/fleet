#!/bin/bash

# Applies the ownership and permissions of Mozilla's .pkg installer (root:admin,
# group-writable) so Firefox can update itself without prompting for its helper tool.

APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")

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
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local was_running

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

  # 'sudo -u' alone doesn't join the user's GUI session, so 'open' can fail silently.
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


MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
if ! sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"; then
	hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
	exit 1
fi
hdiutil detach "$MOUNT_POINT" || { sleep 2; hdiutil detach "$MOUNT_POINT"; } || hdiutil detach -force "$MOUNT_POINT" || true
quit_and_track_application 'org.mozilla.firefox' || exit 1
if [ -d "$APPDIR/Firefox.app" ]; then
	sudo mv "$APPDIR/Firefox.app" "$TMPDIR/Firefox.app.bkp" || exit $?
fi
if ! sudo cp -R "$TMPDIR/Firefox.app" "$APPDIR"; then
	sudo rm -rf "$APPDIR/Firefox.app"
	if [ -d "$TMPDIR/Firefox.app.bkp" ]; then
		sudo mv "$TMPDIR/Firefox.app.bkp" "$APPDIR/Firefox.app"
	fi
	exit 1
fi
if ! sudo chown -R root:admin "$APPDIR/Firefox.app" || ! sudo chmod -R a+rX,a-st,ug+w,o-w "$APPDIR/Firefox.app"; then
	echo "Failed to set ownership and permissions on Firefox.app; Firefox will prompt for its helper tool before it can update."
fi

relaunch_application 'org.mozilla.firefox'
