#!/bin/bash
# CIS 5.6 - Ensure the "root" Account Is Disabled
# Disables the root account and sets its shell to /usr/bin/false.
/usr/bin/sudo /usr/sbin/dsenableroot -d 2>/dev/null || true
/usr/bin/sudo /usr/bin/dscl . -create /Users/root UserShell /usr/bin/false
