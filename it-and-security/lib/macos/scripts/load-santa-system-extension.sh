#!/bin/bash
set -euo pipefail

SANTA_BIN="/Applications/Santa.app/Contents/MacOS/Santa"
if [[ ! -x "$SANTA_BIN" ]]; then
  echo "Santa is not installed at /Applications/Santa.app"
  exit 1
fi

current_user=$(stat -f '%Su' /dev/console 2>/dev/null || echo root)
if [[ "$current_user" != "root" && "$current_user" != "loginwindow" ]]; then
  sudo -u "$current_user" "$SANTA_BIN" --load-system-extension
else
  "$SANTA_BIN" --load-system-extension
fi
