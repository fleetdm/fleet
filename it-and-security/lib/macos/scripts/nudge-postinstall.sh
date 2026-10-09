#!/bin/bash

# Postinstall script to load Nudge LaunchAgent
# This script runs as root, so it loads the LaunchAgent into the logged-in user's GUI domain

PLIST_PATH="/Library/LaunchAgents/com.github.macadmins.Nudge.plist"
LABEL="com.github.macadmins.Nudge"

# Check if the plist file exists
if [[ ! -f "$PLIST_PATH" ]]; then
    echo "Error: LaunchAgent plist not found at $PLIST_PATH"
    exit 1
fi

# Set proper ownership and permissions
/usr/sbin/chown root:wheel "$PLIST_PATH"
/bin/chmod 644 "$PLIST_PATH"

# Find the logged-in user. If no one is logged in, launchd loads the LaunchAgent at the next login.
CONSOLE_USER=$(/usr/bin/stat -f '%Su' /dev/console)
case "$CONSOLE_USER" in
    ""|root|loginwindow|_mbsetupuser)
        echo "No user logged in. LaunchAgent will load at the next login."
        exit 0
        ;;
esac

if ! USER_UID=$(/usr/bin/id -u "$CONSOLE_USER"); then
    echo "Failed to get UID for $CONSOLE_USER"
    exit 1
fi

echo "Loading LaunchAgent for $CONSOLE_USER: $PLIST_PATH"

# Unload first if already loaded
if /bin/launchctl print "gui/$USER_UID/$LABEL" &>/dev/null; then
    echo "LaunchAgent already loaded, unloading first..."
    /bin/launchctl bootout "gui/$USER_UID/$LABEL" 2>/dev/null
fi

# Load the LaunchAgent
if /bin/launchctl bootstrap "gui/$USER_UID" "$PLIST_PATH"; then
    echo "Successfully loaded LaunchAgent"
else
    echo "Failed to load LaunchAgent"
    exit 1
fi

# Verify it's loaded
if /bin/launchctl print "gui/$USER_UID/$LABEL" &>/dev/null; then
    echo "LaunchAgent is now active"
else
    echo "Warning: LaunchAgent may not be properly loaded"
fi

exit 0
