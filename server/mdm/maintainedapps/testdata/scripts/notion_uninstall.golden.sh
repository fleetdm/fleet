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

sudo rm -rf "$APPDIR/Notion.app"
trash $LOGGED_IN_USER '~/Library/Application Support/Caches/notion-updater'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/notion.id.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/Notion'
trash $LOGGED_IN_USER '~/Library/Caches/notion.id*'
trash $LOGGED_IN_USER '~/Library/Logs/Notion'
trash $LOGGED_IN_USER '~/Library/Preferences/ByHost/notion.id.*'
trash $LOGGED_IN_USER '~/Library/Preferences/notion.id.*'
trash $LOGGED_IN_USER '~/Library/Saved Application State/notion.id.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/notion.id'
