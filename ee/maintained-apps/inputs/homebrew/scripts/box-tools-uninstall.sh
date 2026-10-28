#!/bin/bash

# Box Tools installs per-user (~/Library/Application Support/Box/Box Edit), so
# remove it from every local user's home. The parent Box directory is shared
# with other Box products (e.g. Box Drive), so only the Box Edit subdirectory
# is removed; the parent is removed only if it is left empty.

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
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

  # try to quit the application within the timeout period
  local quit_success=false still_running
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1
    sleep 1
    if still_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
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
