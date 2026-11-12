#!/bin/bash
# CIS 2.3.3.1 - Ensure Screen Sharing Is Disabled
# The policy query checks that com.apple.screensharing is disabled
# in /var/db/com.apple.xpc.launchd/disabled.plist.
# The original script disabled com.apple.ODSAgent (DVD/CD Sharing),
# which is a different service and doesn't satisfy the query.

/usr/bin/sudo /bin/launchctl disable system/com.apple.screensharing
