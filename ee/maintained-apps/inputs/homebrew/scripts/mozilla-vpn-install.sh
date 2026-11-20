#!/bin/bash

# Browsers also run this executable as the extension's native messaging host,
# with the manifest path or a chrome-extension:// origin as its argument.
mozilla_vpn_pids() {
  ps -axo pid=,args= \
    | grep -E '^ *[0-9]+ /Applications/Mozilla VPN\.app/Contents/MacOS/Mozilla VPN( |$)' \
    | grep -vE 'mozillavpn\.json|chrome-extension:' \
    | awk '{print $1}'
}

# Mozilla VPN ignores the AppleScript quit. SIGTERM is its own shutdown path:
# it disconnects the VPN, then exits.
quit_mozilla_vpn() {
  local pids
  pids=$(mozilla_vpn_pids)
  [[ -n "$pids" ]] || return 0
  echo "Quitting Mozilla VPN..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..20}; do
    sleep 1
    pids=$(mozilla_vpn_pids)
    if [[ -z "$pids" ]]; then
      echo "Mozilla VPN quit successfully."
      return 0
    fi
  done
  echo "Mozilla VPN didn't quit; forcing it to quit."
  kill -KILL $pids 2>/dev/null
  sleep 1
  if [[ -n "$(mozilla_vpn_pids)" ]]; then
    echo "Mozilla VPN did not quit."
    return 1
  fi
  echo "Mozilla VPN quit."
}

installer_running() {
  kill -0 "$installer_pid" 2>/dev/null
}

wait_for_installer() {
  local seconds="$1" i
  for ((i = 0; i < seconds; i++)); do
    installer_running || return 0
    sleep 1
  done
  ! installer_running
}

# The pkg's postinstall opens the app again.
quit_mozilla_vpn || exit 1

# On some Macs the installer never returns, which would block the host's other
# installs until Fleet kills this script an hour later. The postinstall's last
# step opens the app, so quitting it is tried before giving up.
installer -pkg "$INSTALLER_PATH" -target / &
installer_pid=$!
if ! wait_for_installer 300; then
  echo "The installer hasn't finished after 5 minutes. Quitting the Mozilla VPN app the installer opened..."
  quit_mozilla_vpn
  if ! wait_for_installer 60; then
    echo "The installer is still running. Stopping it."
    kill -TERM "$installer_pid" 2>/dev/null
    exit 1
  fi
fi
wait "$installer_pid"
