#!/bin/bash

for i in $(/usr/bin/sudo dscl . list /Users | grep -v "^_"); do
    /usr/bin/sudo /bin/chmod -R og-rwx /Users/"$i"
done
