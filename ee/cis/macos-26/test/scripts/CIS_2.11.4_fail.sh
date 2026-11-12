#!/bin/bash
# CIS 2.11.4 - Ensure Login Window Displays as Name and Password Is Enabled
# Sets SHOWFULLNAME=false so the query fails.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow SHOWFULLNAME -bool false
