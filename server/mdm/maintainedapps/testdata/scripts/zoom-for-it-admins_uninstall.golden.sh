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

send_signal() {
  local signal="$1"
  local bundle_id="$2"
  local logged_in_user="$3"
  local logged_in_uid pids

  if [ -z "$signal" ] || [ -z "$bundle_id" ] || [ -z "$logged_in_user" ]; then
    echo "Usage: uninstall_signal <signal> <bundle_id> <logged_in_user>"
    return 1
  fi

  logged_in_uid=$(id -u "$logged_in_user")
  if [ -z "$logged_in_uid" ]; then
    echo "Could not find UID for user '$logged_in_user'."
    return 1
  fi

  echo "Signalling '$signal' to application ID '$bundle_id' for user '$logged_in_user'"

  pids=$(/bin/launchctl asuser "$logged_in_uid" sudo -iu "$logged_in_user" /bin/launchctl list | awk -v bundle_id="$bundle_id" '
    $3 ~ bundle_id { print $1 }')

  if [ -z "$pids" ]; then
    echo "No processes found for bundle ID '$bundle_id'."
    return 0
  fi

  echo "Unix PIDs are $pids for processes with bundle identifier $bundle_id"
  for pid in $pids; do
    if kill -s "$signal" "$pid" 2>/dev/null; then
      echo "Successfully signaled PID $pid with signal $signal."
    else
      echo "Failed to kill PID $pid with signal $signal. Check permissions."
    fi
  done

  sleep 3
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

remove_launchctl_service 'us.zoom.ZoomDaemon'
send_signal 'KILL' 'us.zoom.xos' "$LOGGED_IN_USER"
sudo pkgutil --forget 'us.zoom.pkg.videomeeting'
sudo rm -rf '/Applications/zoom.us.app'
sudo rm -rf '/Library/Audio/Plug-Ins/HAL/ZoomAudioDevice.driver'
sudo rm -rf '/Library/Internet Plug-Ins/ZoomUsPlugIn.plugin'
sudo rm -rf '/Library/Logs/DiagnosticReports/zoom.us*'
sudo rm -rf '/Library/PrivilegedHelperTools/us.zoom.ZoomDaemon'
trash $LOGGED_IN_USER '/Library/Preferences/us.zoom.config.plist'
trash $LOGGED_IN_USER '~/.zoomus'
trash $LOGGED_IN_USER '~/Desktop/Zoom'
trash $LOGGED_IN_USER '~/Documents/Zoom'
trash $LOGGED_IN_USER '~/Library/Application Scripts/*.ZoomClient3rd'
trash $LOGGED_IN_USER '~/Library/Application Support/CloudDocs/session/containers/iCloud.us.zoom.videomeetings'
trash $LOGGED_IN_USER '~/Library/Application Support/CloudDocs/session/containers/iCloud.us.zoom.videomeetings.plist'
trash $LOGGED_IN_USER '~/Library/Application Support/CrashReporter/zoom.us*'
trash $LOGGED_IN_USER '~/Library/Application Support/zoom.us'
trash $LOGGED_IN_USER '~/Library/Caches/us.zoom.xos'
trash $LOGGED_IN_USER '~/Library/Cookies/us.zoom.xos.binarycookies'
trash $LOGGED_IN_USER '~/Library/Group Containers/*.ZoomClient3rd'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/us.zoom.xos'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/us.zoom.xos.binarycookies'
trash $LOGGED_IN_USER '~/Library/Internet Plug-Ins/ZoomUsPlugIn.plugin'
trash $LOGGED_IN_USER '~/Library/Logs/zoom.us'
trash $LOGGED_IN_USER '~/Library/Logs/zoominstall.log'
trash $LOGGED_IN_USER '~/Library/Logs/ZoomPhone'
trash $LOGGED_IN_USER '~/Library/Mobile Documents/iCloud~us~zoom~videomeetings'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.airhost.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.caphost.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.Transcode.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.xos.Hotkey.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.xos.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/us.zoom.ZoomAutoUpdater.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/ZoomChat.plist'
trash $LOGGED_IN_USER '~/Library/Safari/PerSiteZoomPreferences.plist'
trash $LOGGED_IN_USER '~/Library/SafariTechnologyPreview/PerSiteZoomPreferences.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/us.zoom.xos.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/us.zoom.xos'
