#!/bin/bash
# CIS 2.11.5 - Ensure Show Password Hints Is Disabled
# Sets RetriesUntilHint=0 so the query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow RetriesUntilHint -int 0
