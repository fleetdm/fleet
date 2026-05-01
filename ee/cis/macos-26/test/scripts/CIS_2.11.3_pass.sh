#!/bin/bash
# CIS 2.11.3 - Ensure a Custom Message for the Login Screen Is Enabled
# Sets a non-empty LoginwindowText so the query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "Authorized use only. Activity may be monitored."
