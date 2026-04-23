#!/bin/bash
# CIS 2.3.3.1 - Ensure Screen Sharing Is Disabled
# Enables the screen sharing launchd service so the policy query fails.
/usr/bin/sudo /bin/launchctl enable system/com.apple.screensharing
/usr/bin/sudo /bin/launchctl bootstrap system /System/Library/LaunchDaemons/com.apple.screensharing.plist
