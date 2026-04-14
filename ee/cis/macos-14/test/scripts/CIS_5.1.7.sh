#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Removes world-write permissions from folders under /Library that
# aren't SIP-protected (don't have the com.apple.rootless xattr).
# The original script had `/usr/bin/sudo IFS=$'\n'` which runs
# IFS in a sudo subshell that exits immediately, leaving IFS unset.
# It also searched /System/Volumes/Data/Library but the query
# checks /Library/%.

IFS=$'\n'

for libPermissions in $( /usr/bin/sudo /usr/bin/find /Library -type d -perm -2 2>/dev/null | /usr/bin/grep -v Caches | /usr/bin/grep -v /Preferences/Audio/Data ); do
    # Skip SIP-protected directories
    if /usr/bin/xattr "$libPermissions" 2>/dev/null | /usr/bin/grep -q "com.apple.rootless"; then
        continue
    fi
    /usr/bin/sudo /bin/chmod -R o-w "$libPermissions"
done
