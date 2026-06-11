#!/bin/bash
# CIS 2.3.2.2 - Ensure the Time Service Is Enabled
# Loads the timed launch daemon so the policy query passes.
/usr/bin/sudo /bin/launchctl load -w /System/Library/LaunchDaemons/com.apple.timed.plist
