#!/bin/bash

# CIS - Ensure No Login Items Exist With Passwords in User Keychain
# Removes password hints for all local user accounts.

for username in $(dscl . -list /Users UniqueID | awk '$2 >= 500 {print $1}'); do
  # Remove the hint attribute if it exists
  if dscl . -read "/Users/$username" hint &>/dev/null; then
    sudo dscl . -delete "/Users/$username" hint
  fi
done
