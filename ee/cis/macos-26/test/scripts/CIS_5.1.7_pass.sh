#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Removes world-write bit from non-sticky, non-rootless directories under /Library.
# Also cleans up the stub directory the _fail.sh script may have created, so it
# doesn't persist across runs.
/usr/bin/sudo /bin/rm -rf /Library/CIS_Test_World_Writable
IFS=$'\n'
for d in $(/usr/bin/find /Library -type d -perm -002 ! -perm -1000 ! -xattrname com.apple.rootless 2>/dev/null); do
  /usr/bin/sudo /bin/chmod -R o-w "$d"
done
