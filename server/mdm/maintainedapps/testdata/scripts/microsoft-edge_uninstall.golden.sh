#!/bin/sh

# variables
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

remove_launchctl_service 'com.microsoft.EdgeUpdater.update-internal.109.0.1518.89.system'
remove_launchctl_service 'com.microsoft.EdgeUpdater.update.system'
remove_launchctl_service 'com.microsoft.EdgeUpdater.wake.system'
sudo pkgutil --forget 'com.microsoft.edgemac'
sudo rm -rf '/Library/Application Support/Microsoft/EdgeUpdater'
sudo rmdir '/Library/Application Support/Microsoft'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.microsoft.edgemac.wdgExtension'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft Edge'
trash $LOGGED_IN_USER '~/Library/Application Support/Microsoft/EdgeUpdater'
trash $LOGGED_IN_USER '~/Library/Caches/com.microsoft.edgemac'
trash $LOGGED_IN_USER '~/Library/Caches/com.microsoft.EdgeUpdater'
trash $LOGGED_IN_USER '~/Library/Caches/Microsoft Edge'
trash $LOGGED_IN_USER '~/Library/Containers/com.microsoft.edgemac.wdgExtension'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.microsoft.edge*'
trash $LOGGED_IN_USER '~/Library/LaunchAgents/com.microsoft.EdgeUpdater.*.plist'
trash $LOGGED_IN_USER '~/Library/Microsoft/EdgeUpdater'
trash $LOGGED_IN_USER '~/Library/Preferences/com.microsoft.edgemac.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.microsoft.edgemac.*'
trash $LOGGED_IN_USER '~/Library/WebKit/com.microsoft.edgemac'
