#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Hot Corners Are Secure For Current User
# Sets one corner to 6 ("Disable Screen Saver") for the console user so the
# query fails. Requires a non-root console user; exits nonzero otherwise so
# the runner does not mistake a silent no-op for a successfully applied fail
# state. cfprefsd is flushed so osquery reads the change from the on-disk plist.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -z "$user" ] || [ "$user" = "root" ]; then
  echo "No non-root console user logged in; cannot apply fail state" >&2
  exit 1
fi
/usr/bin/sudo -u "$user" /usr/bin/defaults write com.apple.dock wvous-br-corner -int 6 || exit 1
/usr/bin/sudo /usr/bin/killall cfprefsd 2>/dev/null || true
