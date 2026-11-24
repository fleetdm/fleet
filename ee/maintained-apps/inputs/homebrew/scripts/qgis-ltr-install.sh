#!/bin/bash

# QGIS LTR is closed if it's running, and unsaved project changes are lost. It
# installs as QGIS-LTR.app so it can sit alongside QGIS, which is left alone.

APPDIR="/Applications"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
APP_PATH="$APPDIR/QGIS-LTR.app"

# QGIS and QGIS LTR share a bundle identifier, so match processes by path.
# Record that QGIS was open so it can be reopened after the install, the way
# relaunch_application expects.
quit_app_at_path() {
  local app_path="${1%/}"
  # pgrep -f and pkill -f take an extended regular expression, so the dots in
  # the bundle name would match any character. Escape them.
  local pattern="^$(printf '%s' "$app_path" | sed 's/[][(){}.^$*+?|\\]/\\&/g')/Contents/MacOS/"
  pgrep -f "$pattern" >/dev/null || return 0
  export APP_WAS_RUNNING_org_qgis_qgis3=1
  echo "Closing $app_path..."
  pkill -TERM -f "$pattern"
  for _ in {1..10}; do
    pgrep -f "$pattern" >/dev/null || return 0
    sleep 1
  done
  echo "$app_path did not close; forcing it to quit."
  pkill -KILL -f "$pattern"
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

MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null || exit 1
trap 'hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || { sleep 2; hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1; } || hdiutil detach -force "$MOUNT_POINT" >/dev/null 2>&1 || true' EXIT

if [[ ! -d "$MOUNT_POINT/QGIS.app" ]]; then
  echo "No QGIS.app found in the disk image."
  exit 1
fi

quit_app_at_path "$APP_PATH"

sudo rm -rf "$TMPDIR/QGIS-LTR.app.bkp"
if [[ -d "$APP_PATH" ]]; then
  sudo mv "$APP_PATH" "$TMPDIR/QGIS-LTR.app.bkp" || exit $?
fi
if ! sudo cp -R "$MOUNT_POINT/QGIS.app" "$APP_PATH"; then
  # remove the partial copy so a failed install isn't inventoried as the new
  # version, then restore the previous version if there was one
  sudo rm -rf "$APP_PATH"
  if [[ -d "$TMPDIR/QGIS-LTR.app.bkp" ]]; then
    sudo mv "$TMPDIR/QGIS-LTR.app.bkp" "$APP_PATH"
  fi
  exit 1
fi
sudo rm -rf "$TMPDIR/QGIS-LTR.app.bkp"

echo "Installed $APP_PATH"

relaunch_application 'org.qgis.qgis3' "$APP_PATH"
