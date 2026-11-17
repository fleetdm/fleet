#!/bin/bash

# The tailscale CLI runs the same executable with arguments.
tailscale_pids() {
  ps -axo pid=,args= \
    | grep -E '^ *[0-9]+ /Applications/Tailscale\.app/Contents/MacOS/Tailscale$' \
    | awk '{print $1}'
}

# Tailscale answers a quit request by asking whether to keep running in the
# menu bar, and a full quit disconnects the VPN. SIGTERM ends the app right
# away and leaves the VPN connected.
quit_tailscale() {
  local pids
  pids=$(tailscale_pids)
  [[ -n "$pids" ]] || return 0
  echo "Quitting Tailscale..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    pids=$(tailscale_pids)
    if [[ -z "$pids" ]]; then
      echo "Tailscale quit successfully."
      return 0
    fi
  done
  echo "Tailscale didn't quit; forcing it to quit."
  kill -KILL $pids 2>/dev/null
  sleep 1
  if [[ -n "$(tailscale_pids)" ]]; then
    echo "Tailscale did not quit."
    return 1
  fi
  echo "Tailscale quit."
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

was_running=false
[[ -n "$(tailscale_pids)" ]] && was_running=true
quit_tailscale || exit 1
installer -pkg "$INSTALLER_PATH" -target / || exit $?
# The pkg's postinstall opens the app; relaunch it only if that didn't.
if [[ "$was_running" == true && -z "$(tailscale_pids)" ]]; then
  export APP_WAS_RUNNING_io_tailscale_ipn_macsys=1
  relaunch_application 'io.tailscale.ipn.macsys'
fi
