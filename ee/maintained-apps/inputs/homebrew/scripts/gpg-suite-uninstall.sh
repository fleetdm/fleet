#!/bin/sh

# variables
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

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

remove_launchctl_service 'org.gpgtools.gpgmail.enable-bundles'
remove_launchctl_service 'org.gpgtools.gpgmail.patch-uuid-user'
remove_launchctl_service 'org.gpgtools.gpgmail.user-uuid-patcher'
remove_launchctl_service 'org.gpgtools.gpgmail.uuid-patcher'
remove_launchctl_service 'org.gpgtools.Libmacgpg.xpc'
remove_launchctl_service 'org.gpgtools.macgpg2.fix'
remove_launchctl_service 'org.gpgtools.macgpg2.gpg-agent'
remove_launchctl_service 'org.gpgtools.macgpg2.shutdown-gpg-agent'
remove_launchctl_service 'org.gpgtools.macgpg2.updater'
remove_launchctl_service 'org.gpgtools.updater'
quit_application 'com.apple.mail'
quit_application 'org.gpgtools.gpgkeychain'
quit_application 'org.gpgtools.gpgkeychainaccess'
quit_application 'org.gpgtools.gpgmail.upgrader'
quit_application 'org.gpgtools.gpgservices'

# Try to find and run GPG Suite's Uninstaller
# GPG Suite installs Uninstall.app in /Library/Application Support/GPGTools/Uninstall.app
# or it might be embedded in the app bundle
UNINSTALLER_PATH=""
if [ -d "/Library/Application Support/GPGTools/Uninstall.app" ]; then
  UNINSTALLER_PATH="/Library/Application Support/GPGTools/Uninstall.app/Contents/Resources/GPG Suite Uninstaller.app/Contents/Resources/uninstall.sh"
elif [ -d "/Applications/GPG Keychain.app/Contents/Resources/Uninstall.app" ]; then
  UNINSTALLER_PATH="/Applications/GPG Keychain.app/Contents/Resources/Uninstall.app/Contents/Resources/GPG Suite Uninstaller.app/Contents/Resources/uninstall.sh"
fi

if [ -n "$UNINSTALLER_PATH" ] && [ -f "$UNINSTALLER_PATH" ]; then
  echo "Running GPG Suite Uninstaller from: $UNINSTALLER_PATH"
  (cd /Users/$LOGGED_IN_USER && sudo "$UNINSTALLER_PATH") || echo "GPG Suite Uninstaller failed, continuing with manual removal"
else
  echo "GPG Suite Uninstaller not found, performing manual removal"
  # Explicitly remove the app bundle to ensure it's gone
  sudo rm -rf '/Applications/GPG Keychain.app'
  sudo rm -rf '/Applications/GPG Mail.app' || true
fi

remove_pkg_files 'org.gpgtools.*'
forget_pkg 'org.gpgtools.*'
sudo rm -rf '/Library/Application Support/GPGTools'
sudo rm -rf '/Library/Frameworks/Libmacgpg.framework'
sudo rm -rf '/Library/Mail/Bundles.gpgmail*'
sudo rm -rf '/Library/Mail/Bundles/GPGMail.mailbundle'
sudo rm -rf '/Library/PreferencePanes/GPGPreferences.prefPane'
sudo rm -rf '/Library/Services/GPGServices.service'
sudo rm -rf '/Network/Library/Mail/Bundles/GPGMail.mailbundle'
sudo rm -rf '/private/etc/manpaths.d/MacGPG2'
sudo rm -rf '/private/etc/paths.d/MacGPG2'
sudo rm -rf '/private/tmp/gpg-agent'
sudo rm -rf '/usr/local/MacGPG2'
trash $LOGGED_IN_USER '~/Library/Application Support/GPGTools'
trash $LOGGED_IN_USER '~/Library/Caches/org.gpgtools.gpg*'
trash $LOGGED_IN_USER '~/Library/Containers/com.apple.mail/Data/Library/Frameworks/Libmacgpg.framework'
trash $LOGGED_IN_USER '~/Library/Containers/com.apple.mail/Data/Library/Preferences/org.gpgtools.*'
trash $LOGGED_IN_USER '~/Library/Frameworks/Libmacgpg.framework'
trash $LOGGED_IN_USER '~/Library/HTTPStorages/org.gpgtools.*'
trash $LOGGED_IN_USER '~/Library/LaunchAgents/org.gpgtools.*'
trash $LOGGED_IN_USER '~/Library/Mail/Bundles/GPGMail.mailbundle'
trash $LOGGED_IN_USER '~/Library/PreferencePanes/GPGPreferences.prefPane'
trash $LOGGED_IN_USER '~/Library/Preferences/org.gpgtools.*'
trash $LOGGED_IN_USER '~/Library/Services/GPGServices.service'

