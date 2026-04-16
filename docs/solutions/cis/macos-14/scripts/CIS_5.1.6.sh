#!/bin/bash

/usr/bin/sudo IFS=$'\n'
for sysPermissions in $( /usr/bin/find /System/Volumes/Data/System -type d -perm -2 | /usr/bin/grep -v "Drop Box" );
do
  /bin/chmod -R o-w "$sysPermissions"
done