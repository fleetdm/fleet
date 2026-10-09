#!/bin/bash

PLIST_PATH="/Library/LaunchAgents/com.github.macadmins.Nudge.plist"
LABEL="com.github.macadmins.Nudge"

if [[ ! -f "$PLIST_PATH" ]]; then
    echo "Error: LaunchAgent plist not found at $PLIST_PATH"
    exit 1
fi

/usr/sbin/chown root:wheel "$PLIST_PATH"
/bin/chmod 644 "$PLIST_PATH"

if /bin/launchctl list | /usr/bin/grep -q "$LABEL"; then
    /bin/launchctl unload "$PLIST_PATH" 2>/dev/null
fi

/bin/launchctl load "$PLIST_PATH" || exit 1
