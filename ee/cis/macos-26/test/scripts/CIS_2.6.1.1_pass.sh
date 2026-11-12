#!/bin/bash
# CIS 2.6.1.1 - Ensure Location Services Is Enabled
# Enables the locationd service and sets LocationServicesEnabled so the query passes.
/usr/bin/sudo /bin/launchctl load -w /System/Library/LaunchDaemons/com.apple.locationd.plist
/usr/bin/sudo /usr/bin/defaults write /var/db/locationd/Library/Preferences/ByHost/com.apple.locationd LocationServicesEnabled -bool true
/usr/bin/sudo /usr/bin/killall locationd
