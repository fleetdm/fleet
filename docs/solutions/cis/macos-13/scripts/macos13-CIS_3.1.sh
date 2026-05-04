#!/bin/bash

/usr/bin/sudo /bin/launchctl load -w /System/Library/LaunchDaemons/com.apple.auditd.plist

# For Testing: After the above command executed:
#   This will stop the service: /usr/bin/sudo /bin/launchctl stop com.apple.auditd
#   This will start the service: /usr/bin/sudo /bin/launchctl start com.apple.auditd