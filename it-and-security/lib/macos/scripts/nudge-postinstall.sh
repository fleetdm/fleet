#!/bin/bash

# Postinstall script to load Nudge LaunchAgent
# This script runs as root

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

echo "Loading LaunchAgent: $PLIST_PATH"

# Check if already loaded and unload first if necessary
if /bin/launchctl list | /usr/bin/grep -q "$LABEL"; then
    echo "LaunchAgent already loaded, unloading first..."
    /bin/launchctl unload "$PLIST_PATH" 2>/dev/null
fi

# Load the LaunchAgent
if /bin/launchctl load "$PLIST_PATH"; then
    echo "Successfully loaded LaunchAgent"
else
    echo "Failed to load LaunchAgent"
    exit 1
fi

# Verify it's loaded
if /bin/launchctl list | /usr/bin/grep -q "$LABEL"; then
    echo "LaunchAgent is now active"
else
    echo "Warning: LaunchAgent may not be properly loaded"
fi

exit 0
