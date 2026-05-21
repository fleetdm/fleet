#!/bin/bash

# CIS - Ensure No World Writable Folders Exist in the Applications Folder

IFS=$'\n'
for apps in $(/usr/bin/find /Applications -iname "*.app" -type d -perm -2); do
  sudo /bin/chmod -R o-w "$apps"
done
unset IFS
