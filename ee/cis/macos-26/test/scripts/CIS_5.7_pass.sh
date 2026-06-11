#!/bin/bash
# CIS 5.7 - Ensure an Administrator Account Cannot Login to Another User's Active and Locked Session
# Sets the system.login.screensaver authorization to require the session owner.
/usr/bin/sudo /usr/bin/security authorizationdb write system.login.screensaver authenticate-session-owner
# Re-enable Touch ID for users.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow screenUnlockMode -int 1
