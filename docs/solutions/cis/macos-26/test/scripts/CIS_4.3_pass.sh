#!/bin/bash
# CIS 4.3 - Ensure NFS Server Is Disabled
# Disables nfsd and removes /etc/exports so the query passes.
/usr/bin/sudo /bin/launchctl disable system/com.apple.nfsd 2>/dev/null || true
/usr/bin/sudo /sbin/nfsd stop 2>/dev/null || true
/usr/bin/sudo /bin/rm -rf /etc/exports
