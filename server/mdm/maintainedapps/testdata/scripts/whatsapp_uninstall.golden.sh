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

sudo rm -rf "$APPDIR/WhatsApp.app"
trash $LOGGED_IN_USER '~/Library/Application Scripts/net.whatsapp.WhatsApp*'
trash $LOGGED_IN_USER '~/Library/Caches/net.whatsapp.WhatsApp'
trash $LOGGED_IN_USER '~/Library/Containers/net.whatsapp.WhatsApp*'
trash $LOGGED_IN_USER '~/Library/Group Containers/group.com.facebook.family'
trash $LOGGED_IN_USER '~/Library/Group Containers/group.net.whatsapp*'
trash $LOGGED_IN_USER '~/Library/Saved Application State/net.whatsapp.WhatsApp.savedState'
