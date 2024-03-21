#!/bin/sh

# Unlock password for all non-root users
awk -F':' '{ if ($3 >= 1000 && $3 < 60000) print $1 }' /etc/passwd | while read user
do
    echo "$user"
    if [ "$user" != "root" ]; then
        echo "Unlocking password for $user"
        passwd -u $user
    fi
done
