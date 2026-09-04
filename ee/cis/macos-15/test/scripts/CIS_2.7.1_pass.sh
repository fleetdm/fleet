#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Hot Corners Are Secure For Current User
# Sets all four hot corners to 0 (no action, != 6) for the console user so
# the query passes. On a headless VM with no non-root console user this
# no-ops. The plist is written by path as root so the script also works
# under Fleet's script runner, where the per-user cfprefsd is unreachable
# (see CIS_2.7.1_fail.sh).
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  plist="/Users/$user/Library/Preferences/com.apple.dock.plist"
  for corner in wvous-tl-corner wvous-tr-corner wvous-bl-corner wvous-br-corner; do
    /usr/bin/sudo /usr/bin/defaults write "$plist" "$corner" -int 0
  done
  /usr/bin/sudo /usr/sbin/chown "$user:$(/usr/bin/id -gn "$user")" "$plist"
  /usr/bin/sudo /usr/bin/killall -u "$user" cfprefsd 2>/dev/null || true
fi
