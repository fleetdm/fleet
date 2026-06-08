#!/bin/bash
# CIS 2.3.2.2 - Ensure the Time Service Is Enabled
# Enables and bootstraps the com.apple.timed launchd service.
/usr/bin/sudo /bin/launchctl enable system/com.apple.timed
/usr/bin/sudo /bin/launchctl bootstrap system /System/Library/LaunchDaemons/com.apple.timed.plist
