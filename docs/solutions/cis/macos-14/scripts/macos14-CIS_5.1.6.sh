#!/bin/bash

# CIS - Ensure No World Writable Folders Exist in the System Folder

IFS=$'\n'
for sysPermissions in $(/usr/bin/find /System/Volumes/Data/System -type d -perm -2 | /usr/bin/grep -v "Drop Box"); do
  sudo /bin/chmod -R o-w "$sysPermissions"
done
unset IFS
