#!/bin/bash
# CIS 4.2 - Ensure HTTP Server Is Disabled
# Stops Apache and unloads the LaunchDaemon so no httpd process is running.
/usr/bin/sudo /usr/sbin/apachectl stop 2>/dev/null || true
/usr/bin/sudo /bin/launchctl unload -w /System/Library/LaunchDaemons/org.apache.httpd.plist 2>/dev/null || true
