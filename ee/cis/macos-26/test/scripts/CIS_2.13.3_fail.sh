#!/bin/bash
# CIS 2.13.3 - Ensure Automatic Login Is Disabled
# Sets autoLoginUser to a placeholder value so the query fails.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow autoLoginUser -string "testuser"
