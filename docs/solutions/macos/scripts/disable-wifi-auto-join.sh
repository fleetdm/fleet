#!/bin/bash

# This script disables auto-join for a specified Wi-Fi network on macOS.
# Must be run as root.
#
# Based on the approach described by Alan Siu:
# https://www.alansiu.net/2026/01/22/scripting-disabling-auto-join-for-wi-fi-networks/

# Replace with the SSID of the Wi-Fi network you want to disable auto-join for.
SSID="CHANGE_ME"

KNOWN_NETWORKS_PLIST="/Library/Preferences/com.apple.wifi.known-networks"

if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root." >&2
    exit 1
fi

if [ "$SSID" = "CHANGE_ME" ]; then
    echo "Error: Please set the SSID variable to the Wi-Fi network name." >&2
    exit 1
fi

echo "Disabling auto-join for Wi-Fi network: $SSID"
/usr/bin/defaults write "$KNOWN_NETWORKS_PLIST" "wifi.network.ssid.$SSID" -dict-add AutoJoinDisabled -bool TRUE

# Verify the change.
auto_join_disabled=$(/usr/bin/defaults read "$KNOWN_NETWORKS_PLIST" "wifi.network.ssid.$SSID" 2>/dev/null | /usr/bin/grep -c "AutoJoinDisabled = 1")

if [ "$auto_join_disabled" -ge 1 ]; then
    echo "Auto-join successfully disabled for $SSID."
else
    echo "Warning: Could not verify auto-join was disabled for $SSID." >&2
    exit 1
fi
