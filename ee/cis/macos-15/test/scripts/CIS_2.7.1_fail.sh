#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Hot Corners Are Secure For Current User
# Sets one corner to 6 ("Disable Screen Saver") for the console user so the
# query fails. Requires a non-root console user; exits nonzero otherwise so
# the runner does not mistake a silent no-op for a successfully applied fail
# state.
# The plist is written by path as root instead of `sudo -u "$user" defaults
# write com.apple.dock`: under Fleet's script runner (a root LaunchDaemon with
# no user session) the per-user cfprefsd is unreachable and the domain write
# fails with "Could not write domain". The on-disk file is what osquery's
# `plist` table reads; the user's cfprefsd is restarted so its stale cache
# cannot clobber it.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -z "$user" ] || [ "$user" = "root" ]; then
  echo "No non-root console user logged in; cannot apply fail state" >&2
  exit 1
fi
plist="/Users/$user/Library/Preferences/com.apple.dock.plist"
/usr/bin/sudo /usr/bin/defaults write "$plist" wvous-br-corner -int 6 || exit 1
/usr/bin/sudo /usr/sbin/chown "$user:$(/usr/bin/id -gn "$user")" "$plist"
/usr/bin/sudo /usr/bin/killall -u "$user" cfprefsd 2>/dev/null || true
