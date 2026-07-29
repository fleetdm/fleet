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

quit_application "com.Box.Box-Edit"
quit_application "com.box.Box-Local-Com-Server"

# Box's background helpers may keep running after a quit attempt (they are
# faceless agents in the user's session); kill any leftovers so the app files
# can be removed cleanly.
pkill -f "Box Edit.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Local Com Server.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Device Trust.app/Contents/MacOS" >/dev/null 2>&1 || true
pkill -f "Box Tools Custom Apps.app/Contents/MacOS" >/dev/null 2>&1 || true

removed=false
for udir in /Users/* /var/root; do
  box_edit_dir="$udir/Library/Application Support/Box/Box Edit"
  [[ -d "$box_edit_dir" ]] || continue
  echo "removing $box_edit_dir"
  rm -rf "$box_edit_dir" || true
  rmdir "$udir/Library/Application Support/Box" >/dev/null 2>&1 || true
  removed=true
done

if [[ "$removed" = false ]]; then
  echo "Box Tools was not found in any user's home directory."
fi

echo "Box Tools uninstalled"
