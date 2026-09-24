#!/bin/bash
# CIS 5.6 - Ensure the "root" Account Is Disabled
# Removes root's secure token, disables the root account, sets its shell to
# /usr/bin/false, then deletes any AuthenticationAuthority value the disable
# step may have re-added — so the key is absent (query reads value = '')
# regardless of OS-specific behavior. `dsenableroot -d` needs an interactive
# admin login and silently no-ops under Fleet's script runner; the `dscl
# delete` is what actually disables root.
/usr/bin/sudo /usr/bin/fdesetup remove -user root 2>/dev/null || true
/usr/bin/sudo /usr/sbin/dsenableroot -d 2>/dev/null || true
/usr/bin/sudo /usr/bin/dscl . -create /Users/root UserShell /usr/bin/false
/usr/bin/sudo /usr/bin/dscl /Local/Default delete /Users/root AuthenticationAuthority 2>/dev/null || true
