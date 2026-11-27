#!/bin/bash
# CIS 2.6.1.2 - Ensure 'Show Location Icon in Control Center when System Services Request Your Location' Is Enabled
# Sets ShowSystemServices=true so the query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.locationmenu.plist ShowSystemServices -bool true
