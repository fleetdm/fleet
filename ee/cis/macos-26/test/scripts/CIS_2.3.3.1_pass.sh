#!/bin/bash
# CIS 2.3.3.1 - Ensure Screen Sharing Is Disabled
# Disables the screen sharing launchd service so the policy query passes.
/usr/bin/sudo /bin/launchctl disable system/com.apple.screensharing
/usr/bin/sudo /bin/launchctl bootout system/com.apple.screensharing
