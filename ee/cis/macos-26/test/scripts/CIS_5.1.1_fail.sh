#!/bin/bash
# CIS 5.1.1 - Ensure Home Folders Are Secure
# Loosens console user's home folder to 755 so the query fails.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ] && [ -d "/Users/$user" ]; then
  /usr/bin/sudo /bin/chmod 755 "/Users/$user"
fi
