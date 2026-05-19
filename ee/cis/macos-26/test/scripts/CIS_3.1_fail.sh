#!/bin/bash
# CIS 3.1 - Ensure Security Auditing Is Enabled
# Unloads auditd so the query fails.
/usr/bin/sudo /bin/launchctl unload -w /System/Library/LaunchDaemons/com.apple.auditd.plist 2>/dev/null || true
