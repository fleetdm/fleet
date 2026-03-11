#!/bin/sh

# Fleet uninstall script for Karabiner-Elements
#
# This custom script is needed because Karabiner-Elements uses an array-of-arrays
# signal format in its homebrew cask definition, which the standard ingester does
# not support. The script mirrors the cask's uninstall stanza:
#   early_script, launchctl, signal, script, pkgutil, delete
# plus the zap stanza (trash).

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
    echo "Usage: send_signal <signal> <bundle_id> <logged_in_user>"
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

expand_pkgid_and_map() {
  local PKGID="$1"
  local FUNC="$2"
  if [[ "$PKGID" == *"*" ]]; then
    local prefix="${PKGID%\*}"
    echo "Expanding wildcard for PKGID: $PKGID"
    for receipt in $(pkgutil --pkgs | grep "^${prefix}"); do
      echo "Processing $receipt"
      "$FUNC" "$receipt"
    done
  else
    "$FUNC" "$PKGID"
  fi
}

forget_pkg() {
  local PKGID="$1"
  expand_pkgid_and_map "$PKGID" forget_receipt
}

forget_receipt() {
  local PKGID="$1"
  sudo pkgutil --forget "$PKGID"
}

remove_pkg_files() {
  local PKGID="$1"
  expand_pkgid_and_map "$PKGID" remove_receipt_files
}

remove_receipt_files() {
  local PKGID="$1"
  local PKGINFO VOLUME INSTALL_LOCATION FULL_INSTALL_LOCATION

  echo "pkgutil --pkg-info-plist \"$PKGID\""
  PKGINFO=$(pkgutil --pkg-info-plist "$PKGID")
  VOLUME=$(echo "$PKGINFO" | awk '/<key>volume<\/key>/ {getline; gsub(/.*<string>|<\/string>.*/, ""); print}')
  INSTALL_LOCATION=$(echo "$PKGINFO" | awk '/<key>install-location<\/key>/ {getline; gsub(/.*<string>|<\/string>.*/, ""); print}')

  if [ -z "$INSTALL_LOCATION" ] || [ "$INSTALL_LOCATION" = "/" ]; then
    FULL_INSTALL_LOCATION="$VOLUME"
  else
    FULL_INSTALL_LOCATION="$VOLUME/$INSTALL_LOCATION"
    FULL_INSTALL_LOCATION=$(echo "$FULL_INSTALL_LOCATION" | sed 's|//|/|g')
  fi

  echo "sudo pkgutil --only-files --files \"$PKGID\" | sed \"s|^|${FULL_INSTALL_LOCATION}/|\" | tr '\\\\n' '\\\\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf"
  sudo pkgutil --only-files --files "$PKGID" | sed "s|^|/${INSTALL_LOCATION}/|" | tr '\n' '\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf

  echo "sudo pkgutil --only-dirs --files \"$PKGID\" | sed \"s|^|${FULL_INSTALL_LOCATION}/|\" | grep '\\.app$' | tr '\\\\n' '\\\\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf"
  sudo pkgutil --only-dirs --files "$PKGID" | sed "s|^|${FULL_INSTALL_LOCATION}/|" | grep '\.app$' | tr '\n' '\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf

  root_app_dir=$(
    sudo pkgutil --only-dirs --files "$PKGID" \
      | sed "s|^|${FULL_INSTALL_LOCATION}/|" \
      | grep 'Applications' \
      | awk '{ print length, $0 }' \
      | sort -n \
      | head -n1 \
      | cut -d' ' -f2-
  )
  if [ -n "$root_app_dir" ]; then
    echo "sudo rmdir -p \"$root_app_dir\" 2>/dev/null || :"
    sudo rmdir -p "$root_app_dir" 2>/dev/null || :
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

# === early_script: run DriverKit VirtualHIDDevice removal ===
if [ -x '/Library/Application Support/org.pqrs/Karabiner-DriverKit-VirtualHIDDevice/scripts/uninstall/remove_files.sh' ]; then
  echo "Running Karabiner DriverKit VirtualHIDDevice early uninstall script..."
  sudo '/Library/Application Support/org.pqrs/Karabiner-DriverKit-VirtualHIDDevice/scripts/uninstall/remove_files.sh'
fi

# === launchctl: remove all Karabiner launch services ===
remove_launchctl_service 'org.pqrs.karabiner.agent.karabiner_grabber'
remove_launchctl_service 'org.pqrs.karabiner.agent.karabiner_observer'
remove_launchctl_service 'org.pqrs.karabiner.karabiner_console_user_server'
remove_launchctl_service 'org.pqrs.karabiner.karabiner_grabber'
remove_launchctl_service 'org.pqrs.karabiner.karabiner_observer'
remove_launchctl_service 'org.pqrs.karabiner.karabiner_session_monitor'
remove_launchctl_service 'org.pqrs.karabiner.NotificationWindow'

# === signal: send TERM to Karabiner menu bar and notification processes ===
send_signal 'TERM' 'org.pqrs.Karabiner-Menu' "$LOGGED_IN_USER"
send_signal 'TERM' 'org.pqrs.Karabiner-NotificationWindow' "$LOGGED_IN_USER"

# === script: run Karabiner's own uninstall_core.sh ===
if [ -x '/Library/Application Support/org.pqrs/Karabiner-Elements/uninstall_core.sh' ]; then
  (cd /Users/$LOGGED_IN_USER && sudo '/Library/Application Support/org.pqrs/Karabiner-Elements/uninstall_core.sh')
fi

# === pkgutil: remove package receipts and files ===
remove_pkg_files 'org.pqrs.Karabiner-DriverKit-VirtualHIDDevice'
forget_pkg 'org.pqrs.Karabiner-DriverKit-VirtualHIDDevice'
remove_pkg_files 'org.pqrs.Karabiner-Elements'
forget_pkg 'org.pqrs.Karabiner-Elements'

# === delete: remove support directory ===
sudo rm -rf '/Library/Application Support/org.pqrs'

# === zap: trash user-specific files ===
trash $LOGGED_IN_USER '~/.config/karabiner'
trash $LOGGED_IN_USER '~/.local/share/karabiner'
trash $LOGGED_IN_USER '~/Library/Application Scripts/org.pqrs.Karabiner-VirtualHIDDevice-Manager'
trash $LOGGED_IN_USER '~/Library/Application Support/Karabiner-Elements'
trash $LOGGED_IN_USER '~/Library/Caches/org.pqrs.Karabiner-Elements.Updater'
trash $LOGGED_IN_USER '~/Library/Containers/org.pqrs.Karabiner-VirtualHIDDevice-Manager'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/org.pqrs.Karabiner-Elements.Settings'
trash $LOGGED_IN_USER '~/Library/Preferences/org.pqrs.Karabiner-Elements.Updater.plist'
