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

quit_application 'org.mozilla.firefox'
sudo rm -rf "$APPDIR/Firefox.app"
sudo rm -rf 'firefox'
sudo rmdir '~/Library/Application Support/Mozilla'
sudo rmdir '~/Library/Caches/Mozilla'
sudo rmdir '~/Library/Caches/Mozilla/updates'
sudo rmdir '~/Library/Caches/Mozilla/updates/Applications'
trash $LOGGED_IN_USER '/Library/Logs/DiagnosticReports/firefox_*'
trash $LOGGED_IN_USER '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/org.mozilla.firefox.sfl*'
trash $LOGGED_IN_USER '~/Library/Application Support/CrashReporter/firefox_*'
trash $LOGGED_IN_USER '~/Library/Application Support/Firefox'
trash $LOGGED_IN_USER '~/Library/Caches/Firefox'
trash $LOGGED_IN_USER '~/Library/Caches/Mozilla/updates/Applications/Firefox'
trash $LOGGED_IN_USER '~/Library/Caches/org.mozilla.crashreporter'
trash $LOGGED_IN_USER '~/Library/Caches/org.mozilla.firefox'
trash $LOGGED_IN_USER '~/Library/Preferences/org.mozilla.crashreporter.plist'
trash $LOGGED_IN_USER '~/Library/Preferences/org.mozilla.firefox.plist'
trash $LOGGED_IN_USER '~/Library/Saved Application State/org.mozilla.firefox.savedState'
trash $LOGGED_IN_USER '~/Library/WebKit/org.mozilla.firefox'
