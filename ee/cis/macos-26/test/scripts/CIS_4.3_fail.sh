#!/bin/bash
# CIS 4.3 - Ensure NFS Server Is Disabled
# Creates /etc/exports and starts nfsd so the query fails.
echo "# test exports for CIS 4.3 regression" | /usr/bin/sudo /usr/bin/tee /etc/exports > /dev/null
/usr/bin/sudo /bin/launchctl enable system/com.apple.nfsd 2>/dev/null || true
/usr/bin/sudo /sbin/nfsd start 2>/dev/null || true
