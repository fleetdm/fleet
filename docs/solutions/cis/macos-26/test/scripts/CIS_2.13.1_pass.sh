#!/bin/bash
# CIS 2.13.1 - Ensure Guest Account Is Disabled
# Sets GuestEnabled=false in the loginwindow plist so the query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow GuestEnabled -bool false
