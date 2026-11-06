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

# The pkg's postinstall opens the app again.
quit_mozilla_vpn || exit 1
installer -pkg "$INSTALLER_PATH" -target / || exit $?
