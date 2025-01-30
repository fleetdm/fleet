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

remove_launchctl_service 'com.docker.helper'
remove_launchctl_service 'com.docker.socket'
remove_launchctl_service 'com.docker.vmnetd'
quit_application 'com.docker.docker'
quit_application 'com.electron.dockerdesktop'
sudo rm -rf '/Library/PrivilegedHelperTools/com.docker.socket'
sudo rm -rf '/Library/PrivilegedHelperTools/com.docker.vmnetd'
sudo rmdir '~/.docker/bin'
sudo rm -rf "$APPDIR/Docker.app"
sudo rm -rf '/usr/local/bin/docker'
sudo rm -rf '/usr/local/bin/docker-credential-desktop'
sudo rm -rf '/usr/local/bin/docker-credential-ecr-login'
sudo rm -rf '/usr/local/bin/docker-credential-osxkeychain'
sudo rm -rf '/usr/local/bin/docker-index'
sudo rm -rf '/usr/local/bin/kubectl.docker'
sudo rm -rf '/usr/local/cli-plugins/docker-compose'
sudo rm -rf '/usr/local/bin/hub-tool'
sudo rmdir '~/Library/Caches/com.plausiblelabs.crashreporter.data'
sudo rmdir '~/Library/Caches/KSCrashReports'
trash $LOGGED_IN_USER '/usr/local/bin/docker-compose.backup'
trash $LOGGED_IN_USER '/usr/local/bin/docker.backup'
trash $LOGGED_IN_USER '~/.docker'
trash $LOGGED_IN_USER '~/Library/Application Scripts/com.docker.helper'
trash $LOGGED_IN_USER '~/Library/Application Scripts/group.com.docker'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.docker.helper.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.electron.dockerdesktop.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/com.bugsnag.Bugsnag/com.docker.docker'
trash $LOGGED_IN_USER '~/Library/Application Support/Docker Desktop'
trash $LOGGED_IN_USER '~/Library/Caches/com.docker.docker'
trash $LOGGED_IN_USER '~/Library/Caches/com.plausiblelabs.crashreporter.data/com.docker.docker'
trash $LOGGED_IN_USER '~/Library/Caches/KSCrashReports/Docker'
trash $LOGGED_IN_USER '~/Library/Containers/com.docker.docker'
trash $LOGGED_IN_USER '~/Library/Containers/com.docker.helper'
trash $LOGGED_IN_USER '~/Library/Group Containers/group.com.docker'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.docker.docker'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/com.docker.docker.binarycookies'
trash $LOGGED_IN_USER '~/Library/Logs/Docker Desktop'
trash $LOGGED_IN_USER '~/Library/Preferences/com.docker.docker.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/com.electron.docker-frontend.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/com.electron.dockerdesktop.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.electron.docker-frontend.savedState'
trash $LOGGED_IN_USER '~/Library/Saved Application State/com.electron.dockerdesktop.savedState'
