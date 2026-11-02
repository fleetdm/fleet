#!/bin/bash

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"
  local timeout_duration=10

  # check if the application is running
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

  # App was running, mark it for relaunch
  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

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


# Root removes the client-tools cache from a pinned working directory after
# checking its kernel path (/bin/pwd, not the shell's string walk): the home's
# owner controls ~/.tsh, so root following a symlink they swapped in could
# otherwise delete any "bin" directory on the system.
remove_tsh_bin() { # remove_tsh_bin <home directory>
  local tsh_dir="$1/.tsh" real_home
  real_home=$(cd "$1" && /bin/pwd -P) || return 1
  (
    cd "$tsh_dir" || exit 1
    if [[ "$(/bin/pwd -P)" != "$real_home/.tsh" ]]; then
      echo "refusing to remove $tsh_dir/bin: $tsh_dir resolves to $(/bin/pwd -P)"
      exit 1
    fi
    rm -rf bin
  )
}


# install pkg files
quit_and_track_application 'com.gravitational.teleport.tsh' || exit 1
sudo installer -pkg "$INSTALLER_PATH" -target / || exit $?

# Teleport's client-tools auto-updater keeps several older tsh/tctl versions
# under ~/.tsh/bin/<uuid>-update-pkg-v2/*.pkg/Payload/*.app. osquery inventories
# those bundles, so they keep the patch policy failing after an update. Remove
# the cache (config included, since tsh trusts it without checking the files);
# tsh rebuilds it on demand.
cache_failed=false
for home in /Users/*; do
  [[ -d "$home/.tsh/bin" ]] || continue
  echo "Removing Teleport client-tools cache '$home/.tsh/bin'."
  if ! remove_tsh_bin "$home"; then
    echo "Failed to remove '$home/.tsh/bin'."
    cache_failed=true
  fi
done

relaunch_application 'com.gravitational.teleport.tsh'
[[ "$cache_failed" = false ]] || exit 1
