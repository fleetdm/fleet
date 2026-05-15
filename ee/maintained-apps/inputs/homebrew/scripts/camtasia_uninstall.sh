#!/bin/bash

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

sudo rm -rf "$APPDIR/Camtasia.app"
trash $LOGGED_IN_USER '/Users/Shared/TechSmith/Camtasia'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.techsmith.camtasia26.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.techsmith.camtasia2026.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/TechSmith/Camtasia*'
trash $LOGGED_IN_USER '~/Library/Caches/com.plausiblelabs.crashreporter.data/com.techsmith.camtasia*'
trash $LOGGED_IN_USER '~/Library/Caches/com.techsmith.camtasia*'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.techsmith.camtasia*'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.techsmith.camtasia*.binarycookies'
trash $LOGGED_IN_USER '~/Library/Preferences/com.techsmith.camtasia*.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.techsmith.camtasia*.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/com.techsmith.camtasia*'
