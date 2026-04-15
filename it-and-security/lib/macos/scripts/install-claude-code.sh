#!/bin/bash
set -e

# Install Claude Code using the official installer
# https://code.claude.com/docs/en/quickstart
# Fleet runs scripts as root; install as the logged-in user.

# Determine the logged-in (console) user
CURRENT_USER=$(/usr/bin/stat -f%Su /dev/console)

if [ "$CURRENT_USER" = "root" ] || [ -z "$CURRENT_USER" ]; then
  echo "Error: unable to determine the logged-in user."
  exit 1
fi

# Install Claude Code as the logged-in user using the official installer
/usr/bin/sudo -u "$CURRENT_USER" bash -c 'curl -fsSL https://claude.ai/install.sh | sh'

echo "Claude Code installed successfully."
exit 0
