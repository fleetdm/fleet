#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Mirrors the canonical resolution in cis-policy-queries.yml: let `find`
# exclude sticky-bit directories (! -perm -1000) and SIP-protected
# directories (! -xattrname com.apple.rootless) directly, so the
# filter matches the query exactly.
# Previous versions piped through `grep -v Caches | grep -v
# /Preferences/Audio/Data`, which did a substring match anywhere in
# the path and wasn't aligned with what the query actually checks.

IFS=$'\n'
for libPermissions in $(/usr/bin/sudo /usr/bin/find /Library -type d -perm -002 ! -perm -1000 ! -xattrname com.apple.rootless 2>/dev/null); do
    /usr/bin/sudo /bin/chmod -R o-w "$libPermissions"
done
