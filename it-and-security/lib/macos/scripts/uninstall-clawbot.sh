#!/bin/bash

# Uninstall Clawbot from macOS

if [ -d "/Applications/Clawbot.app" ]; then
  rm -rf "/Applications/Clawbot.app"
fi

# Check user Applications folders
for user_dir in /Users/*/Applications/Clawbot.app; do
  if [ -d "$user_dir" ]; then
    rm -rf "$user_dir"
  fi
done

exit 0
