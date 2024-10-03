#!/bin/sh

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

trash() {
  local logged_in_user="$1"
  local target_file="$2"
  local timestamp="$(date +%Y-%m-%d-%s)"

  # replace ~ with /Users/$logged_in_user
  if [[ "$target_file" == ~* ]]; then
    target_file="/Users/$logged_in_user${target_file:1}"
  fi

  local trash="/Users/$logged_in_user/.Trash"
  local file_name="$(basename "${target_file}")"

  if [[ -e "$target_file" ]]; then
    echo "removing $target_file."
    mv -f "$target_file" "$trash/${file_name}_${timestamp}"
  else
    echo "$target_file doesn't exist."
  fi
}

sudo rm -rf "$APPDIR/Postman.app"
trash $LOGGED_IN_USER '~/Library/Application Support/com.postmanlabs.mac.ShipIt'
trash $LOGGED_IN_USER '~/Library/Application Support/Postman'
trash $LOGGED_IN_USER '~/Library/Caches/com.postmanlabs.mac'
trash $LOGGED_IN_USER '~/Library/Caches/com.postmanlabs.mac.ShipIt'
trash $LOGGED_IN_USER '~/Library/Caches/Postman'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.postmanlabs.mac'
trash $LOGGED_IN_USER '~/Library/Preferences/ByHost/com.postmanlabs.mac.ShipIt.*.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/com.postmanlabs.mac.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.postmanlabs.mac.savedState'
