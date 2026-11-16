#!/bin/bash

# Box Tools is a per-user application: Box only supports installing it into a
# user's home directory (~/Library/Application Support/Box/Box Edit). Its admin
# .pkg forbids the local system domain (enable_localSystem="false") and Box's
# large-scale deployment docs instruct running the installer as the console
# user. This script replicates the Homebrew cask install: it copies the app
# bundles out of the DMG's "Install Box Tools.app" into the console user's
# home, then registers them with LaunchServices so osquery's apps table (which
# enumerates LaunchServices) and the box.com web app can find them.

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
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
    return 1
  fi
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

# An interrupted run of the earlier root-based installer can leave Box, Box
# Edit or the bundles inside root-owned, which the user-run steps below can't
# touch. Root repairs that from a pinned working directory after checking its
# kernel path (/bin/pwd, not the shell's string walk), so a symlink the user
# swaps in can't redirect it. Leftover bundles are unlinked, never chowned
# recursively, because a recursive chown would follow planted hard links.
repair_legacy_ownership() { # repair_legacy_ownership <dir> <expected kernel path> [bundle...]
  local dir="$1" expected="$2"
  shift 2
  [[ -d "$dir" ]] || return 0
  (
    cd "$dir" || exit 1
    if [[ "$(/bin/pwd -P)" != "$expected" ]]; then
      echo "refusing to repair $dir: it resolves to $(/bin/pwd -P)"
      exit 1
    fi
    if [[ "$(stat -f %u .)" -eq 0 ]]; then
      chown "$target_user":staff . || exit 1
    fi
    local app
    for app in "$@"; do
      if [[ -e "$app" && "$(stat -f %u "$app")" != "$target_uid" ]]; then
        rm -rf "$app" || exit 1
      fi
    done
    exit 0
  )
}

[[ -n "$INSTALLER_PATH" && -f "$INSTALLER_PATH" ]] || { echo "missing installer"; exit 1; }

target_user=$(stat -f "%Su" /dev/console)
if [[ -z "$target_user" || "$target_user" == "root" || "$target_user" == "loginwindow" || "$target_user" == "_mbsetupuser" ]]; then
  # No GUI session (e.g. install triggered while logged out): fall back to the
  # last user that logged in. Box only supports Box Tools on single-user Macs,
  # so this is unambiguous in the supported configuration.
  target_user=$(defaults read /Library/Preferences/com.apple.loginwindow lastUserName 2>/dev/null)
fi
if [[ -z "$target_user" || "$target_user" == "root" ]] || ! id -u "$target_user" >/dev/null 2>&1; then
  echo "Box Tools installs per-user; no logged-in (or last logged-in) user found."
  exit 1
fi
target_uid=$(id -u "$target_user")

user_home=$(dscl . -read "/Users/$target_user" NFSHomeDirectory 2>/dev/null | sed 's/^NFSHomeDirectory: //')
[[ -n "$user_home" ]] || user_home="/Users/$target_user"
[[ -d "$user_home" ]] || { echo "home directory for $target_user not found"; exit 1; }

quit_and_track_application "com.Box.Box-Edit" || exit 1
quit_and_track_application "com.box.Box-Local-Com-Server" || exit 1

BOX_DIR="$user_home/Library/Application Support/Box"
DEST="$BOX_DIR/Box Edit"
BUNDLES=("Box Edit.app" "Box Device Trust.app" "Box Local Com Server.app" "Box Tools Custom Apps.app")
real_home=$(cd "$user_home" && /bin/pwd -P) || exit 1
repair_legacy_ownership "$BOX_DIR" "$real_home/Library/Application Support/Box" || exit 1
repair_legacy_ownership "$DEST" "$real_home/Library/Application Support/Box/Box Edit" "${BUNDLES[@]}" || exit 1

MOUNT_POINT="$(mktemp -d /tmp/box-tools-dmg.XXXXXX)"
hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null || { echo "failed to mount dmg"; exit 1; }

RESOURCES="$MOUNT_POINT/Install Box Tools.app/Contents/Resources"

# Create and copy as the target user, never as root: the user controls this
# tree, so root following a planted symlink out of it could create or chown
# files anywhere on the system. Files the user copies are already theirs.
status=0
if sudo -u "$target_user" mkdir -p "$DEST"; then
  for app in "${BUNDLES[@]}"; do
    if [[ -d "$RESOURCES/$app" ]]; then
      if ! sudo -u "$target_user" rm -rf "${DEST:?}/$app" || ! sudo -u "$target_user" ditto "$RESOURCES/$app" "$DEST/$app"; then
        echo "failed to copy $app"
        status=1
      fi
    else
      echo "$app not found in installer"
      status=1
    fi
  done
else
  echo "failed to create $DEST"
  status=1
fi

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || { sleep 2; hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1; } || hdiutil detach -force "$MOUNT_POINT" >/dev/null 2>&1 || true
rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true

[[ $status -eq 0 ]] || exit "$status"

# Register the copied bundles with LaunchServices in both the root context
# (osqueryd runs as root and its apps table enumerates LaunchServices) and the
# user's context (so box.com can launch Box Edit without a first manual launch).
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
for app in "${BUNDLES[@]}"; do
  "$LSREGISTER" -f "$DEST/$app" >/dev/null 2>&1 || true
  /bin/launchctl asuser "$target_uid" sudo -u "$target_user" "$LSREGISTER" -f "$DEST/$app" >/dev/null 2>&1 || true
done

relaunch_application "com.Box.Box-Edit"
relaunch_application "com.box.Box-Local-Com-Server"

echo "Box Tools installed for user $target_user"
