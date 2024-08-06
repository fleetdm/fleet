#!/bin/sh

# Unlock password for all non-root users
awk -F':' '{ if ($3 >= 1000 && $3 < 60000) print $1 }' /etc/passwd | while read user
do
    echo "$user"
    if [ "$user" != "root" ]; then
        echo "Unlocking password for $user"
        STDERR=$(passwd -u "$user" 2>&1 >/dev/null)
        if [ $? -eq 3 ]; then
          # possibly due to the user not having a password
          # use this convoluted case approach to avoid bashisms (POSIX portable)
          case "$STDERR" in
            *"unlocking the password would result in a passwordless account"* )
              # unlock and delete password to set it back to empty
              passwd -ud "$user"
            ;;
          esac
        fi
    fi
done

# Remove the pam_nologin file
rm /etc/nologin
