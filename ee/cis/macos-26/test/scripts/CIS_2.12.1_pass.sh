#!/bin/bash
# CIS 2.12.1 - Ensure Users' Accounts Do Not Have a Password Hint
# Removes the hint attribute from every local user so the query passes.
users=$(/usr/bin/sudo /usr/bin/dscl . -list /Users hint 2>/dev/null | /usr/bin/awk 'NF==2 {print $1}')
for u in $users; do
  /usr/bin/sudo /usr/bin/dscl . -delete "/Users/$u" hint 2>/dev/null || true
done
