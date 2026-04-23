#!/bin/bash
# CIS 2.13.1 - Ensure Guest Account Is Disabled
# Enables the guest account so the query fails.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow GuestEnabled -bool true
