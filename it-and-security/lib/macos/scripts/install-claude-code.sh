#!/bin/bash
set -e

# Install Claude Code via Homebrew
# Fleet runs scripts as root; brew must run as the logged-in user.

# Determine the logged-in (console) user
CURRENT_USER=$(/usr/bin/stat -f%Su /dev/console)

if [ "$CURRENT_USER" = "root" ] || [ -z "$CURRENT_USER" ]; then
  echo "Error: unable to determine the logged-in user."
  exit 1
fi

CURRENT_USER_HOME=$(/usr/bin/dscl . -read "/Users/$CURRENT_USER" NFSHomeDirectory | awk '{print $2}')

# Locate the Homebrew binary
if [ -x "/opt/homebrew/bin/brew" ]; then
  BREW_PATH="/opt/homebrew/bin/brew"
elif [ -x "/usr/local/bin/brew" ]; then
  BREW_PATH="/usr/local/bin/brew"
else
  echo "Error: Homebrew is not installed. Please install Homebrew first."
  exit 1
fi

# Check if Claude Code is already installed
if /usr/bin/sudo -u "$CURRENT_USER" "$BREW_PATH" list claude-code &> /dev/null; then
  echo "Claude Code is already installed. Upgrading..."
  /usr/bin/sudo -u "$CURRENT_USER" "$BREW_PATH" upgrade claude-code || true
else
  /usr/bin/sudo -u "$CURRENT_USER" "$BREW_PATH" install claude-code
fi

echo "Claude Code installed successfully."
exit 0
