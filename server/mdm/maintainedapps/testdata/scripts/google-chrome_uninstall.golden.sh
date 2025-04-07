#!/bin/sh

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

remove_launchctl_service() {
  local service="$1"
  local booleans=("true" "false")
  local plist_status
  local paths
  local should_sudo

  echo "Removing launchctl service ${service}"

  for should_sudo in "${booleans[@]}"; do
    plist_status=$(launchctl list "${service}" 2>/dev/null)

    if [[ $plist_status == \{* ]]; then
      if [[ $should_sudo == "true" ]]; then
        sudo launchctl remove "${service}"
      else
        launchctl remove "${service}"
      fi
      sleep 1
    fi

    paths=(
      "/Library/LaunchAgents/${service}.plist"
      "/Library/LaunchDaemons/${service}.plist"
    )

    # if not using sudo, prepend the home directory to the paths
    if [[ $should_sudo == "false" ]]; then
      for i in "${!paths[@]}"; do
        paths[i]="${HOME}${paths[i]}"
      done
    fi

    for path in "${paths[@]}"; do
      if [[ -e "$path" ]]; then
        if [[ $should_sudo == "true" ]]; then
          sudo rm -f -- "$path"
        else
          rm -f -- "$path"
        fi
      fi
    done
  done
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

sudo rm -rf "$APPDIR/Google Chrome.app"
remove_launchctl_service 'com.google.keystone.agent'
remove_launchctl_service 'com.google.keystone.daemon'
sudo rmdir '/Library/Google'
sudo rmdir '~/Library/Application Support/Google'
sudo rmdir '~/Library/Caches/Google'
sudo rmdir '~/Library/Google'
trash $LOGGED_IN_USER '/Library/Caches/com.google.SoftwareUpdate.*'
trash $LOGGED_IN_USER '/Library/Google/Google Chrome Brand.plist'
trash $LOGGED_IN_USER '/Library/Google/GoogleSoftwareUpdate'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.google.chrome.app.*.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.google.chrome.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/Google/Chrome'
trash $LOGGED_IN_USER '~/Library/Caches/com.google.Chrome'
trash $LOGGED_IN_USER '~/Library/Caches/com.google.Chrome.helper.*'
trash $LOGGED_IN_USER '~/Library/Caches/com.google.Keystone'
trash $LOGGED_IN_USER '~/Library/Caches/com.google.Keystone.Agent'
trash $LOGGED_IN_USER '~/Library/Caches/com.google.SoftwareUpdate'
trash $LOGGED_IN_USER '~/Library/Caches/Google/Chrome'
trash $LOGGED_IN_USER '~/Library/Google/Google Chrome Brand.plist'
trash $LOGGED_IN_USER '~/Library/Google/GoogleSoftwareUpdate'
trash $LOGGED_IN_USER '~/Library/LaunchAgents/com.google.keystone.agent.plist'
trash $LOGGED_IN_USER '~/Library/LaunchAgents/com.google.keystone.xpcservice.plist'
trash $LOGGED_IN_USER '~/Library/Logs/GoogleSoftwareUpdateAgent.log'
trash $LOGGED_IN_USER '~/Library/Preferences/com.google.Chrome.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/com.google.Keystone.Agent.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.google.Chrome.app.*.savedState'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.google.Chrome.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/com.google.Chrome'
