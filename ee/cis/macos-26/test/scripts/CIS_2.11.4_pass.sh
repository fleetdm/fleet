#!/bin/bash
# CIS 2.11.4 - Ensure Login Window Displays as Name and Password Is Enabled
# Sets SHOWFULLNAME=true so the query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow SHOWFULLNAME -bool true
