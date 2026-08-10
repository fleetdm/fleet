#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Hot Corners Are Secure For Current User
# Sets all four hot corners to 0 (no action, != 6) for the console user so
# the query passes. On a headless VM with no non-root console user this
# no-ops. cfprefsd is flushed so osquery reads the change from the on-disk plist.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  for corner in wvous-tl-corner wvous-tr-corner wvous-bl-corner wvous-br-corner; do
    /usr/bin/sudo -u "$user" /usr/bin/defaults write com.apple.dock "$corner" -int 0
  done
  # defaults writes buffer in cfprefsd; force a flush to the on-disk plist.
  /usr/bin/sudo /usr/bin/killall cfprefsd 2>/dev/null || true
fi
