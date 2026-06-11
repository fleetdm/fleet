#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Corners Are Secure
# Sets one corner to 6 ("Disable Screen Saver") on the console user so the query fails.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  /usr/bin/sudo -u "$user" /usr/bin/defaults write com.apple.dock wvous-br-corner -int 6
fi
