#!/bin/bash
# CIS 5.10 - Ensure XProtect Is Running and Updated
# Loads both XProtect launch daemons and triggers an update.
/usr/bin/sudo /bin/launchctl load -w /Library/Apple/System/Library/LaunchDaemons/com.apple.XProtect.daemon.scan.plist 2>/dev/null || true
/usr/bin/sudo /bin/launchctl load -w /Library/Apple/System/Library/LaunchDaemons/com.apple.XprotectFramework.PluginService.plist 2>/dev/null || true
/usr/bin/sudo /usr/bin/xprotect update 2>/dev/null || true
