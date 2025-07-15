#!/bin/sh
# Script to uninstall Beekeeper Studio from macOS
# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

trash() {
  local logged_in_user="$1"
  local target_file="$2"
  local timestamp="$(date +%Y-%m-%d-%s)"
  local rand="$(jot -r 1 0 99999)"

  # replace ~ with /Users/$logged_in_user
  if [[ "$target_file" == ~* ]]; then
    target_file="/Users/$logged_in_user${target_file:1}"
  fi

  local trash="/Users/$logged_in_user/.Trash"
  local file_name="$(basename "${target_file}")"

  if [[ -e "$target_file" ]]; then
    echo "removing $target_file."
    mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}"
  else
    echo "$target_file doesn't exist."
  fi
}

sudo rm -rf "$APPDIR/Beekeeper Studio.app"
trash $LOGGED_IN_USER '~/Library/Application Support/beekeeper-studio'
trash $LOGGED_IN_USER '~/Library/Application Support/Caches/beekeeper-studio-updater'
trash $LOGGED_IN_USER '~/Library/Caches/io.beekeeperstudio.desktop'
trash $LOGGED_IN_USER '~/Library/Caches/io.beekeeperstudio.desktop.ShipIt'
trash $LOGGED_IN_USER '~/Library/Preferences/ByHost/io.beekeeperstudio.desktop.ShipIt.*.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/io.beekeeperstudio.desktop.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/io.beekeeperstudio.desktop.savedState'

