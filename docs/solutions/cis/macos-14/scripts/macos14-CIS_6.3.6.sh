#!/bin/bash

# CIS - Ensure Safari Advertising Privacy Protection Is Enabled
# Applies privateClickMeasurementEnabled for all local user accounts.

for username in $(dscl . -list /Users UniqueID | awk '$2 >= 500 {print $1}'); do
  home_dir=$(dscl . -read "/Users/$username" NFSHomeDirectory 2>/dev/null | awk '{print $2}')
  pref_path="$home_dir/Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari"
  if [ -d "$home_dir" ]; then
    /usr/bin/sudo -u "$username" /usr/bin/defaults write "$pref_path" WebKitPreferences.privateClickMeasurementEnabled -bool true
  fi
done
