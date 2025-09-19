#!/bin/bash
# Please don't delete. This script is used in the guide here: https://fleetdm.com/guides/scripts

if [ "$EUID" -ne 0 ]; then
  echo "This script requires administrator privileges. Please run with sudo."
  exit 1
fi
# Enable scrippts in Orbit environment variables (plist)
/usr/libexec/PlistBuddy -c "set EnvironmentVariables:ORBIT_ENABLE_SCRIPTS true" "/Library/LaunchDaemons/com.fleetdm.orbit.plist"
# Stop Orbit, wait for stop to complete, and then restart.
launchctl bootout system/com.fleetdm.orbit
while pgrep orbit > /dev/null; do sleep 1 ; done
launchctl bootstrap system $plist_path
echo "Fleet script execution has been enabled and Orbit restarted."
