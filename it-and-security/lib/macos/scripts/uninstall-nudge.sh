#!/bin/bash
# Removes Nudge, its LaunchAgent, and Fleet's Nudge assets.

LABEL="com.github.macadmins.Nudge"

console_user=$(/usr/bin/stat -f '%Su' /dev/console)
if [[ -n "$console_user" && "$console_user" != "root" && "$console_user" != "loginwindow" ]]; then
    uid=$(/usr/bin/id -u "$console_user") && /bin/launchctl bootout "gui/$uid/$LABEL" 2>/dev/null
fi
/usr/bin/pkill -x Nudge

/bin/rm -f "/Library/LaunchAgents/$LABEL.plist"
/bin/rm -rf /Applications/Utilities/Nudge.app /private/var/fleet/nudge-assets

for pkg in com.github.macadmins.Nudge com.fleetdm.Nudge.assets Nudgeassets; do
    /usr/sbin/pkgutil --pkg-info "$pkg" >/dev/null 2>&1 && /usr/sbin/pkgutil --forget "$pkg" >/dev/null
done

if [[ -e /Applications/Utilities/Nudge.app || -e "/Library/LaunchAgents/$LABEL.plist" ]]; then
    echo "Nudge is still present."
    exit 1
fi
echo "Nudge removed."
