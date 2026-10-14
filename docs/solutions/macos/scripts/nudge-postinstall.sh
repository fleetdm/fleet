#!/bin/bash

# Used in the Nudge guide: https://fleetdm.com/guides/enforce-os-updates-on-macos-with-nudge
# If you change this script, update the guide too.

PLIST_PATH="/Library/LaunchAgents/com.github.macadmins.Nudge.plist"
LABEL="com.github.macadmins.Nudge"

if [[ ! -f "$PLIST_PATH" ]]; then
    echo "Error: LaunchAgent plist not found at $PLIST_PATH"
    exit 1
fi

/usr/sbin/chown root:wheel "$PLIST_PATH"
/bin/chmod 644 "$PLIST_PATH"

# Post-install scripts run as root, so load the LaunchAgent into the logged-in user's GUI session.
# If no one is logged in, launchd loads it at the next login.
CONSOLE_USER=$(/usr/bin/stat -f '%Su' /dev/console)
case "$CONSOLE_USER" in
    ""|root|loginwindow|_mbsetupuser)
        echo "No user logged in. The LaunchAgent will load at the next login."
        exit 0
        ;;
esac

USER_UID=$(/usr/bin/id -u "$CONSOLE_USER") || exit 1

/bin/launchctl bootout "gui/$USER_UID/$LABEL" 2>/dev/null
/bin/launchctl bootstrap "gui/$USER_UID" "$PLIST_PATH" || exit 1
