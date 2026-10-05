#!/bin/bash

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1
    sleep 1
    if [[ "$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)" != "true" ]]; then
      echo "Application '$bundle_id' quit successfully."
      quit_success=true
      break
    fi
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
    return 1
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

  # If the target contains glob characters, expand it and move each match.
  if [[ "$target_file" == *[*?[]* ]]; then
    local file file_name
    local matched=false
    local i=0
    # compgen -G expands the (quoted) pattern itself, so paths containing
    # spaces glob correctly; reading line by line keeps each match intact.
    while IFS= read -r file; do
      [[ -n "$file" ]] || continue
      [[ -e "$file" || -L "$file" ]] || continue
      matched=true
      i=$((i + 1))
      file_name="$(basename "$file")"
      echo "removing $file."
      # The per-match counter keeps matches that share a basename from
      # overwriting each other in the trash.
      mv -f "$file" "$trash/${file_name}_${timestamp}_${rand}_${i}"
    done < <(compgen -G "$target_file" 2>/dev/null)
    if [[ "$matched" == false ]]; then
      echo "$target_file doesn't exist."
    fi
    return
  fi

  local file_name="$(basename "${target_file}")"

  if [[ -e "$target_file" ]]; then
    echo "removing $target_file."
    mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}"
  else
    echo "$target_file doesn't exist."
  fi
}

# KiCad is a folder of apps in /Applications/KiCad; each one is quit before
# the folder is removed, and nothing is removed if one won't quit.
# org.kicad-pcb.* is the bundle ID prefix older KiCad releases used.
for app in "$APPDIR/KiCad"/*.app; do
  [[ -d "$app" ]] || continue
  bundle_id=$(/usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$app/Contents/Info.plist" 2>/dev/null)
  [[ "$bundle_id" =~ ^[A-Za-z0-9._-]+$ ]] || continue
  quit_application "$bundle_id" || exit 1
done
sudo rm -rf "$APPDIR/KiCad" || exit $?
sudo rm -rf '/Library/Application Support/kicad' || exit $?
# ~/Documents/KiCad is left in place: it's KiCad's default projects folder.
trash $LOGGED_IN_USER '~/Library/Application Support/kicad'
trash $LOGGED_IN_USER '~/Library/Caches/kicad'
trash $LOGGED_IN_USER '~/Library/Preferences/kicad'
trash $LOGGED_IN_USER '~/Library/Preferences/org.kicad.*'
trash $LOGGED_IN_USER '~/Library/Preferences/org.kicad-pcb.*'
trash $LOGGED_IN_USER '~/Library/Saved Application State/org.kicad.*'
trash $LOGGED_IN_USER '~/Library/Saved Application State/org.kicad-pcb.*'
