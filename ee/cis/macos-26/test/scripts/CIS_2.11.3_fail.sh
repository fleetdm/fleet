#!/bin/bash
# CIS 2.11.3 - Ensure a Custom Message for the Login Screen Is Enabled
# Removes LoginwindowText so the query fails.
/usr/bin/sudo /usr/bin/defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || true
