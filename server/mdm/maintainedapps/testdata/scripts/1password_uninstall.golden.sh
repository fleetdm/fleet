#!/bin/sh

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

sudo rm -rf "$APPDIR/1Password.app"
trash $LOGGED_IN_USER '~/Library/Application Scripts/2BUA8C4S2C.com.1password*'
trash $LOGGED_IN_USER '~/Library/Application Scripts/2BUA8C4S2C.com.agilebits'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.1password.1password-launcher'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.1password.browser-support'
trash $LOGGED_IN_USER '~/Library/Application Support/1Password'
trash $LOGGED_IN_USER '~/Library/Application Support/Arc/User Data/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.1password.1password.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/CrashReporter/1Password*'
trash $LOGGED_IN_USER '~/Library/Application Support/Google/Chrome Beta/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Google/Chrome Canary/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Google/Chrome Dev/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft Edge Beta/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft Edge Canary/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft Edge Dev/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft Edge/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Mozilla/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Application Support/Vivaldi/NativeMessagingHosts/com.1password.1password.json'
trash $LOGGED_IN_USER '~/Library/Containers/2BUA8C4S2C.com.1password.browser-helper'
trash $LOGGED_IN_USER '~/Library/Containers/com.1password.1password*'
trash $LOGGED_IN_USER '~/Library/Containers/com.1password.browser-support'
trash $LOGGED_IN_USER '~/Library/Group Containers/2BUA8C4S2C.com.1password'
trash $LOGGED_IN_USER '~/Library/Group Containers/2BUA8C4S2C.com.agilebits'
trash $LOGGED_IN_USER '~/Library/Logs/1Password'
trash $LOGGED_IN_USER '~/Library/Preferences/com.1password.1password.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/group.com.1password.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.1password.1password.savedState'
