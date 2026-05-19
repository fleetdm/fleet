#!/bin/bash
# CIS 2.13.3 - Ensure Automatic Login Is Disabled
# Removes autoLoginUser so the query passes.
/usr/bin/sudo /usr/bin/defaults delete /Library/Preferences/com.apple.loginwindow autoLoginUser 2>/dev/null || true
