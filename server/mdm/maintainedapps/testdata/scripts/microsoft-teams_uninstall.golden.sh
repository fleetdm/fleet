#!/bin/sh

# variables
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


remove_launchctl_service() {
  local service="$1"
  local booleans=("true" "false")
  local plist_status
  local paths
  local sudo

  echo "Removing launchctl service ${service}"

  for sudo in "${booleans[@]}"; do
    plist_status=$(launchctl list "${service}" 2>/dev/null)

    if [[ $plist_status == \{* ]]; then
      if [[ $sudo == "true" ]]; then
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
    if [[ $sudo == "false" ]]; then
      for i in "${!paths[@]}"; do
        paths[i]="${HOME}${paths[i]}"
      done
    fi

    for path in "${paths[@]}"; do
      if [[ -e "$path" ]]; then
        if [[ $sudo == "true" ]]; then
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

remove_launchctl_service 'com.microsoft.teams.TeamsUpdaterDaemon'
quit_application 'com.microsoft.autoupdate2'
sudo pkgutil --forget 'com.microsoft.MSTeamsAudioDevice'
sudo pkgutil --forget 'com.microsoft.package.Microsoft_AutoUpdate.app'
sudo pkgutil --forget 'com.microsoft.teams2'
sudo rm -rf '/Applications/Microsoft Teams.app'
sudo rm -rf '/Library/Application Support/Microsoft/TeamsUpdaterDaemon'
sudo rm -rf '/Library/Logs/Microsoft/MSTeams'
sudo rm -rf '/Library/Logs/Microsoft/Teams'
sudo rm -rf '/Library/Preferences/com.microsoft.teams.plist'
sudo rmdir '~/Library/Application Support/Microsoft'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.microsoft.teams2'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.microsoft.teams2.launcher'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.microsoft.teams2.notificationcenter'
trash $LOGGED_IN_USER '~/Library/Application Support/com.microsoft.teams'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft/Teams'
trash $LOGGED_IN_USER '~/Library/Application Support/Teams'
trash $LOGGED_IN_USER '~/Library/Caches/com.microsoft.teams'
trash $LOGGED_IN_USER '~/Library/Containers/com.microsoft.teams2'
trash $LOGGED_IN_USER '~/Library/Containers/com.microsoft.teams2.launcher'
trash $LOGGED_IN_USER '~/Library/Containers/com.microsoft.teams2.notificationcenter'
trash $LOGGED_IN_USER '~/Library/Cookies/com.microsoft.teams.binarycookies'
trash $LOGGED_IN_USER '~/Library/Group Containers/*.com.microsoft.teams'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.microsoft.teams'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.microsoft.teams.binarycookies'
trash $LOGGED_IN_USER '~/Library/Logs/Microsoft Teams Helper (Renderer)'
trash $LOGGED_IN_USER '~/Library/Logs/Microsoft Teams'
trash $LOGGED_IN_USER '~/Library/Preferences/com.microsoft.teams.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.microsoft.teams.savedState'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.microsoft.teams2.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/com.microsoft.teams'
