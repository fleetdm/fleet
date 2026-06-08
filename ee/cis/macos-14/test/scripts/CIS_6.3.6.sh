#!/bin/bash
# CIS 6.3.6 - Ensure Advertising Privacy Protection in Safari Is Enabled
# Enables privateClickMeasurement for each non-system user account.
# The original script had literal <username> placeholders that were
# never substituted.
# Note: This policy requires Full Disk Access on the fleetd agent
# to read the Safari container. Without FDA the query can't see
# the plist regardless of its value.

for user in $(/usr/bin/dscl . -list /Users UniqueID | /usr/bin/awk '$2 >= 500 { print $1 }'); do
    home="/Users/$user"
    safari_dir="$home/Library/Containers/com.apple.Safari/Data/Library/Preferences"
    if [ -d "$safari_dir" ]; then
        /usr/bin/sudo -u "$user" /usr/bin/defaults write \
            "$safari_dir/com.apple.Safari" \
            WebKitPreferences.privateClickMeasurementEnabled -bool true
    fi
done
