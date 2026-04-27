#!/bin/bash
# CIS 5.1.5 - Ensure Appropriate Permissions Are Enabled for System Wide Applications
# Removes world-write bit from every .app in /Applications so the query passes.
# Also cleans up the stub bundle the _fail.sh script may have created, so it
# doesn't persist across runs.
/usr/bin/sudo /bin/rm -rf /Applications/CIS_Test_World_Writable.app
IFS=$'\n'
for app in $(/usr/bin/find /Applications -iname "*.app" -type d -perm -2 2>/dev/null); do
  /usr/bin/sudo /bin/chmod -R o-w "$app"
done
