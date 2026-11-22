#!/bin/bash

# Box Tools installs per-user (~/Library/Application Support/Box/Box Edit), so
# remove it from every local user's home. The parent Box directory is shared
# with other Box products (e.g. Box Drive), so only the Box Edit subdirectory
# is removed; the parent is removed only if it is left empty.

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

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  local app_running
  app_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

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

quit_application "com.Box.Box-Edit"
quit_application "com.box.Box-Local-Com-Server"

# Box's background helpers may keep running after a quit attempt (they are
# faceless agents in the user's session); kill any leftovers so the app files
# can be removed cleanly.
pkill -f "Box Edit.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Local Com Server.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Device Trust.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Tools Custom Apps.app/Contents/MacOS" >/dev/null 2>&1 || true

# Root removes Box Edit from a pinned working directory after checking its
# kernel path (/bin/pwd, not the shell's string walk): the home's owner
# controls this tree, so root following a symlink they swapped in could
# otherwise delete files anywhere on the system.
remove_box_edit() { # remove_box_edit <home directory>
  local box_dir="$1/Library/Application Support/Box" real_home
  real_home=$(cd "$1" && /bin/pwd -P) || return 1
  (
    cd "$box_dir" || exit 1
    if [[ "$(/bin/pwd -P)" != "$real_home/Library/Application Support/Box" ]]; then
      echo "refusing to remove $box_dir/Box Edit: $box_dir resolves to $(/bin/pwd -P)"
      exit 1
    fi
    rm -rf "Box Edit" || exit 1
    # Box is shared with Box Drive, so it goes only if Box Edit was all it held.
    cd -P .. && rmdir Box >/dev/null 2>&1
    exit 0
  )
}

removed=false
failed=false
for udir in /Users/* /var/root; do
  box_edit_dir="$udir/Library/Application Support/Box/Box Edit"
  [[ -d "$box_edit_dir" ]] || continue
  echo "removing $box_edit_dir"
  if remove_box_edit "$udir"; then
    removed=true
  else
    echo "failed to remove $box_edit_dir"
    failed=true
  fi
done

if [[ "$removed" = false && "$failed" = false ]]; then
  echo "Box Tools was not found in any user's home directory."
fi

[[ "$failed" = false ]] || exit 1

echo "Box Tools uninstalled"
