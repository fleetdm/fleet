#!/bin/bash

# CIS - Ensure Show All Filename Extensions Setting is Enabled
# Applies AppleShowAllExtensions for all local user accounts.

for username in $(dscl . -list /Users UniqueID | awk '$2 >= 500 {print $1}'); do
  home_dir=$(dscl . -read "/Users/$username" NFSHomeDirectory 2>/dev/null | awk '{print $2}')
  if [ -d "$home_dir" ]; then
    /usr/bin/sudo -u "$username" /usr/bin/defaults write "$home_dir/Library/Preferences/.GlobalPreferences.plist" AppleShowAllExtensions -bool true
  fi
done
