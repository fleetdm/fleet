#!/bin/bash
# CIS 5.1.6 - Ensure No World Writable Folders Exist in the System Folder
# Removes world-write bit from any directory found under /System/Volumes/Data/System.
IFS=$'\n'
for d in $(/usr/bin/find /System/Volumes/Data/System -type d -perm -2 2>/dev/null | /usr/bin/grep -vE "downloadDir|locks"); do
  /usr/bin/sudo /bin/chmod -R o-w "$d"
done
