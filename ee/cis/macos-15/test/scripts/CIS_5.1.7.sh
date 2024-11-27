#!/bin/bash

/usr/bin/sudo IFS=$'\n'
for libPermissions in $( /usr/bin/find /System/Volumes/Data/Library -type d -perm -2 | /usr/bin/grep -v Caches | /usr/bin/grep -v /Preferences/Audio/Data);
do
  /bin/chmod -R o-w "$libPermissions"
done
