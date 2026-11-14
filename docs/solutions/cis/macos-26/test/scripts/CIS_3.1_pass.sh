#!/bin/bash
# CIS 3.1 - Ensure Security Auditing Is Enabled
# Loads auditd and stages the audit_control file so the query passes.
if [ ! -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/cp /etc/security/audit_control.example /etc/security/audit_control
fi
/usr/bin/sudo /bin/launchctl load -w /System/Library/LaunchDaemons/com.apple.auditd.plist
