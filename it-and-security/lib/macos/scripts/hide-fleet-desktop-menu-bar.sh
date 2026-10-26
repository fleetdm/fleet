#!/bin/sh
# Hide the Fleet Desktop menu bar icon on Macs that have Fleet Desktop.app.
# Sets ORBIT_FLEET_DESKTOP to false, then reloads orbit from a temporary LaunchDaemon,
# because orbit runs this script and booting it out directly would kill the script too.

ORBIT_PLIST="/Library/LaunchDaemons/com.fleetdm.orbit.plist"
RELOAD_LABEL="com.fleetdm.hide-fleet-desktop-menu-bar"

if [ ! -f "$ORBIT_PLIST" ]; then
    echo "Fleet's agent (fleetd) is not installed."
    exit 1
fi

/usr/bin/plutil -replace EnvironmentVariables.ORBIT_FLEET_DESKTOP -string "false" "$ORBIT_PLIST" || exit 1

dir=$(/usr/bin/mktemp -d /private/tmp/fleet-hide-desktop.XXXXXX) || exit 1

cat << EOF > "$dir/reload.sh"
#!/bin/sh
/bin/sleep 15
/bin/launchctl bootout system "$ORBIT_PLIST"
/usr/bin/pkill -x fleet-desktop
count=0
until /bin/launchctl bootstrap system "$ORBIT_PLIST"; do
    count=\$((count + 1))
    [ "\$count" -ge 30 ] && break
    /bin/sleep 1
done
/bin/launchctl kickstart "system/com.fleetdm.orbit"
/bin/rm -rf "$dir"
/bin/launchctl bootout "system/$RELOAD_LABEL"
EOF

cat << EOF > "$dir/reload.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$RELOAD_LABEL</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>$dir/reload.sh</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
EOF

if ! /bin/launchctl bootstrap system "$dir/reload.plist"; then
    /bin/rm -rf "$dir"
    echo "Couldn't schedule the fleetd reload. Is one already pending?"
    exit 1
fi

echo "Fleet Desktop menu bar icon will be hidden in about 15 seconds."
