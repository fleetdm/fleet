#!/bin/bash
# CIS 2.12.1 - Ensure Users' Accounts Do Not Have a Password Hint
# Sets a hint attribute on the invoking console user so the query fails.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  /usr/bin/sudo /usr/bin/dscl . -create "/Users/$user" hint "test hint"
fi
