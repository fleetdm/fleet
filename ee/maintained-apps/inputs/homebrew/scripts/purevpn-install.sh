#!/bin/bash

purevpn_pids() {
  ps -axo pid=,args= \
    | grep -E '^ *[0-9]+ /Applications/PureVPN\.app/Contents/MacOS/PureVPN( |$)' \
    | awk '{print $1}'
}

# PureVPN answers a quit request with a "disconnect and quit?" prompt while the
# VPN is connected or connecting. It has no SIGTERM handler, so SIGTERM ends the
# app right away, as the pkg's preinstall does. The VPN tunnel runs in a system
# extension, which isn't signalled.
quit_purevpn() {
  local pids
  pids=$(purevpn_pids)
  [[ -n "$pids" ]] || return 0
  echo "Quitting PureVPN..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    pids=$(purevpn_pids)
    if [[ -z "$pids" ]]; then
      echo "PureVPN quit successfully."
      return 0
    fi
  done
  echo "PureVPN didn't quit; forcing it to quit."
  kill -KILL $pids 2>/dev/null
  sleep 1
  if [[ -n "$(purevpn_pids)" ]]; then
    echo "PureVPN did not quit."
    return 1
  fi
  echo "PureVPN quit."
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
[[ -n "$(purevpn_pids)" ]] && was_running=true
quit_purevpn || exit 1
installer -pkg "$INSTALLER_PATH" -target / || exit $?
# The pkg's postinstall opens the app; relaunch it only if that didn't.
if [[ "$was_running" == true && -z "$(purevpn_pids)" ]]; then
  export APP_WAS_RUNNING_com_purevpn_app_mac=1
  relaunch_application 'com.purevpn.app.mac'
fi
