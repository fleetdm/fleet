#!/bin/bash

# Box Tools is a per-user application: Box only supports installing it into a
# user's home directory (~/Library/Application Support/Box/Box Edit). Its admin
# .pkg forbids the local system domain (enable_localSystem="false") and Box's
# large-scale deployment docs instruct running the installer as the console
# user. This script replicates the Homebrew cask install: it copies the app
# bundles out of the DMG's "Install Box Tools.app" into the console user's
# home, then registers them with LaunchServices so osquery's apps table (which
# enumerates LaunchServices) and the box.com web app can find them.

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

quit_application "com.Box.Box-Edit"
quit_application "com.box.Box-Local-Com-Server"

MOUNT_POINT="$(mktemp -d /tmp/box-tools-dmg.XXXXXX)"
hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null || { echo "failed to mount dmg"; exit 1; }

RESOURCES="$MOUNT_POINT/Install Box Tools.app/Contents/Resources"
BOX_DIR="$user_home/Library/Application Support/Box"
DEST="$BOX_DIR/Box Edit"
mkdir -p "$DEST"

status=0
for app in "Box Edit.app" "Box Device Trust.app" "Box Local Com Server.app" "Box Tools Custom Apps.app"; do
  if [[ -d "$RESOURCES/$app" ]]; then
    rm -rf "${DEST:?}/$app"
    if ! ditto "$RESOURCES/$app" "$DEST/$app"; then
      echo "failed to copy $app"
      status=1
    fi
  else
    echo "$app not found in installer"
    status=1
  fi
done

# The parent Box directory is shared with other Box products (e.g. Box Drive)
# that run as the same user, so owning it (non-recursively) is safe.
chown -R "$target_user":staff "$DEST"
chown "$target_user":staff "$BOX_DIR"

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true

[[ $status -eq 0 ]] || exit "$status"

# Register the copied bundles with LaunchServices in both the root context
# (osqueryd runs as root and its apps table enumerates LaunchServices) and the
# user's context (so box.com can launch Box Edit without a first manual launch).
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
for app in "Box Edit.app" "Box Device Trust.app" "Box Local Com Server.app" "Box Tools Custom Apps.app"; do
  "$LSREGISTER" -f "$DEST/$app" >/dev/null 2>&1 || true
  /bin/launchctl asuser "$target_uid" sudo -u "$target_user" "$LSREGISTER" -f "$DEST/$app" >/dev/null 2>&1 || true
done

echo "Box Tools installed for user $target_user"
