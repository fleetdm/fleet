#!/bin/sh

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}


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

quit_application 'com.tinyspeck.slackmacgap'
sudo rm -rf "$APPDIR/Slack.app"
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.tinyspeck.slackmacgap'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.tinyspeck.slackmacgap.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/Slack'
trash $LOGGED_IN_USER '~/Library/Caches/com.tinyspeck.slackmacgap*'
trash $LOGGED_IN_USER '~/Library/Containers/com.tinyspeck.slackmacgap*'
trash $LOGGED_IN_USER '~/Library/Cookies/com.tinyspeck.slackmacgap.binarycookies'
trash $LOGGED_IN_USER '~/Library/Group Containers/*.com.tinyspeck.slackmacgap'
trash $LOGGED_IN_USER '~/Library/Group Containers/*.slack'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.tinyspeck.slackmacgap*'
trash $LOGGED_IN_USER '~/Library/Logs/Slack'
trash $LOGGED_IN_USER '~/Library/Preferences/ByHost/com.tinyspeck.slackmacgap.ShipIt.*.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/com.tinyspeck.slackmacgap*'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.tinyspeck.slackmacgap.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/com.tinyspeck.slackmacgap'
