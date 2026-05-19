#!/bin/bash

# Custom install script for OpenVPN Connect on macOS.

set -u

APPDIR="/Applications"
BUNDLE_ID="org.openvpn.client.app"
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
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

relaunch_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
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

if [ -z "${INSTALLER_PATH:-}" ] || [ ! -f "$INSTALLER_PATH" ]; then
  echo "Missing or invalid INSTALLER_PATH"
  exit 1
fi

MOUNT_POINT=$(mktemp -d /tmp/openvpn_connect_dmg.XXXXXX)
cleanup() {
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if ! hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null; then
  echo "Failed to mount DMG at $INSTALLER_PATH"
  exit 1
fi

# Locate the arm64 installer pkg. We only support Apple Silicon for this FMA,
# so deliberately skip the x86_64 pkg shipped in the same DMG. Use a glob so
# parentheses and version/build numbers in the file name don't have to be
# hard-coded here.
PKG=""
for candidate in "$MOUNT_POINT"/*_arm64_Installer_signed.pkg; do
  if [ -e "$candidate" ]; then
    PKG="$candidate"
    break
  fi
done

if [ -z "$PKG" ] || [ ! -e "$PKG" ]; then
  echo "Could not find an arm64 OpenVPN Connect installer pkg in the DMG. Contents:"
  ls -la "$MOUNT_POINT"
  exit 1
fi

echo "Installing $PKG..."

quit_and_track_application "$BUNDLE_ID"

if ! sudo installer -pkg "$PKG" -target /; then
  echo "installer -pkg failed for $PKG"
  exit 1
fi

cleanup
trap - EXIT

# OpenVPN Connect 3.8+ places the .app inside a wrapper directory rather than
# directly under /Applications/. osquery's apps table doesn't recurse into
# /Applications/, so it depends on LaunchServices to find nested .app bundles.
# Force-register the installed app with LaunchServices so it shows up in
# osquery's `apps` table immediately.
if [ -x "$LSREGISTER" ]; then
  if [ -d "$APPDIR/OpenVPN Connect/OpenVPN Connect.app" ]; then
    "$LSREGISTER" -f "$APPDIR/OpenVPN Connect/OpenVPN Connect.app" >/dev/null 2>&1 || true
  elif [ -d "$APPDIR/OpenVPN Connect.app" ]; then
    "$LSREGISTER" -f "$APPDIR/OpenVPN Connect.app" >/dev/null 2>&1 || true
  else
    # As a last resort, recursively register anything OpenVPN Connect-shaped
    # under /Applications/ so LaunchServices and osquery can find it.
    "$LSREGISTER" -R -f "$APPDIR/OpenVPN Connect" >/dev/null 2>&1 || true
  fi
fi

relaunch_application "$BUNDLE_ID"

echo "OpenVPN Connect installed"
