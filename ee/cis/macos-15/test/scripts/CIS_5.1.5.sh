#!/bin/bash

/usr/bin/sudo IFS=$'\n'
for apps in $( /usr/bin/find /Applications -iname "*\.app" -type d -perm -2 );
do
  /bin/chmod -R o-w "$apps"
done