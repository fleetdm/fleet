#!/bin/bash

# CIS - Ensure No World Writable Folders Exist in the Library Folder

IFS=$'\n'
for libPermissions in $(/usr/bin/find /System/Volumes/Data/Library -type d -perm -2 | /usr/bin/grep -v Caches | /usr/bin/grep -v /Preferences/Audio/Data); do
  sudo /bin/chmod -R o-w "$libPermissions"
done
unset IFS
