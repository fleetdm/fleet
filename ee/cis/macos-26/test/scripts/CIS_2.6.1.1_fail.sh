#!/bin/bash
# CIS 2.6.1.1 - Ensure Location Services Is Enabled
# Disables Location Services so the query fails.
/usr/bin/sudo /usr/bin/defaults write /var/db/locationd/Library/Preferences/ByHost/com.apple.locationd LocationServicesEnabled -bool false
/usr/bin/sudo /usr/bin/killall locationd
