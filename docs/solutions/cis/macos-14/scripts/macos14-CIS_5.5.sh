#!/bin/bash

# CIS - Ensure a Separate Timestamp Is Used for Each User-tty Combo
# Sets sudo timestamp_type to tty (per-terminal authentication).

SUDOERS_FILE="/etc/sudoers.d/CIS_55_sudoconfiguration"

echo 'Defaults timestamp_type=tty' | sudo tee "$SUDOERS_FILE" > /dev/null
sudo /bin/chmod 0440 "$SUDOERS_FILE"
sudo /usr/sbin/chown root:wheel "$SUDOERS_FILE"

# Validate syntax
if ! sudo /usr/sbin/visudo -cf "$SUDOERS_FILE"; then
  echo "ERROR: sudoers syntax check failed. Removing invalid configuration."
  sudo /bin/rm -f "$SUDOERS_FILE"
  exit 1
fi
