#!/bin/bash
# CIS 2.3.2.2 - Ensure the Time Service Is Enabled
# Unloads the timed launch daemon so the policy query fails.
/usr/bin/sudo /bin/launchctl unload -w /System/Library/LaunchDaemons/com.apple.timed.plist
