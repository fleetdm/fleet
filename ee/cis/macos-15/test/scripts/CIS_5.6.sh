#!/bin/bash
# CIS 5.6 - Ensure the "root" Account Is Disabled
# Removes root's secure token (if any) and disables the root user so the
# AuthenticationAuthority key is absent and the query passes.
/usr/bin/sudo /usr/bin/fdesetup remove -user root 2>/dev/null || true
/usr/bin/sudo /usr/bin/dscl /Local/Default delete /Users/root AuthenticationAuthority 2>/dev/null || true
/usr/bin/sudo /usr/sbin/dsenableroot -d
