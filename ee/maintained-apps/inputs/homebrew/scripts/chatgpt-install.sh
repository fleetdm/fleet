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


is_app_owner() {
  [[ -n "$1" && "$1" != "root" && "$1" != "loginwindow" && "$1" != "_mbsetupuser" ]] && id -u "$1" >/dev/null 2>&1
}

# extract contents
ditto -x -k "$INSTALLER_PATH" "$TMPDIR" || exit 1
# copy to the applications folder
# ChatGPT reports com.openai.codex since OpenAI rebuilt it on the Codex app;
# older installs report com.openai.chat. Quit only the app being replaced, so a
# running ChatGPT Classic or Codex.app with the other ID stays open.
CURRENT_BUNDLE_ID=$(defaults read "${APPDIR}ChatGPT.app/Contents/Info" CFBundleIdentifier 2>/dev/null)
if [[ "$CURRENT_BUNDLE_ID" == "com.openai.codex" ]]; then
	quit_and_track_application 'com.openai.codex' || exit 1
elif [[ "$CURRENT_BUNDLE_ID" == "com.openai.chat" ]]; then
	quit_and_track_application 'com.openai.chat' || exit 1
fi
# Preserve the existing user owner when updating a shared Mac.
target_user=$(stat -f "%Su" "$APPDIR/ChatGPT.app" 2>/dev/null)
if ! is_app_owner "$target_user"; then
	target_user=$(stat -f "%Su" /dev/console)
fi
if ! is_app_owner "$target_user"; then
	target_user=$(defaults read /Library/Preferences/com.apple.loginwindow lastUserName 2>/dev/null)
fi
if [ -d "$APPDIR/ChatGPT.app" ]; then
	sudo mv "$APPDIR/ChatGPT.app" "$TMPDIR/ChatGPT.app.bkp" || exit $?
fi
if ! sudo cp -R "$TMPDIR/ChatGPT.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/ChatGPT.app"
	if [ -d "$TMPDIR/ChatGPT.app.bkp" ]; then
		sudo mv "$TMPDIR/ChatGPT.app.bkp" "$APPDIR/ChatGPT.app"
	fi
	exit 1
fi
# Sparkle must be able to preserve bundle ownership, not just write its files.
if is_app_owner "$target_user"; then
	if ! sudo chown -R "$target_user":staff "$APPDIR/ChatGPT.app"; then
		echo "Failed to assign ownership of ChatGPT.app to '$target_user'; its updater may request administrator authentication."
	fi
else
	echo "No existing owner or logged-in (or last logged-in) user found; ChatGPT.app stays owned by root and its updater may request administrator authentication."
fi

# If the older app was running, open the new one in its place. Register it with
# the user's LaunchServices first, or opening it by bundle ID fails.
if [[ "$APP_WAS_RUNNING_com_openai_chat" == "1" ]]; then
	console_user=$(stat -f "%Su" /dev/console)
	/bin/launchctl asuser "$(id -u "$console_user")" sudo -u "$console_user" /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "${APPDIR}ChatGPT.app" >/dev/null 2>&1 || true
	export APP_WAS_RUNNING_com_openai_codex=1
fi
relaunch_application 'com.openai.codex' "$APPDIR/ChatGPT.app"
