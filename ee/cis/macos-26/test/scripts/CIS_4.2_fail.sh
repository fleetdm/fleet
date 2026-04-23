#!/bin/bash
# CIS 4.2 - Ensure HTTP Server Is Disabled
# Loads Apache and starts httpd so the query fails.
/usr/bin/sudo /bin/launchctl load -w /System/Library/LaunchDaemons/org.apache.httpd.plist 2>/dev/null || true
/usr/bin/sudo /usr/sbin/apachectl start 2>/dev/null || true
