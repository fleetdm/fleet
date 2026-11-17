#!/bin/bash

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
  return 1
}


quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  local app_running
  app_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)
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

  local quit_success=false still_running quit_error
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    quit_error=$(osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" 2>&1 >/dev/null)
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
      echo "Application '$bundle_id' isn't running."
      quit_success=true
      break
    fi
    sleep 1
    if still_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
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


prism_pids() {
  local uid out pid puid exe
  uid=$(id -u "$1") || return 1
  out=$(ps -axo pid=,uid=,comm=) || return 1
  while read -r pid puid exe; do
    [[ "$puid" == "$uid" && ( "$exe" == */Contents/MacOS/Prism || "$exe" == */Contents/MacOS/Logger ) ]] || continue
    if [[ "$(defaults read "${exe%/Contents/MacOS/*}/Contents/Info" CFBundleIdentifier 2>/dev/null)" == "com.GraphPad.Prism" ]]; then
      echo "$pid"
    fi
  done <<< "$out"
}

# Prism doesn't always answer the AppleScript quit. It has no SIGTERM handler,
# so SIGTERM ends it right away, along with its Logger helper. Changes made
# since Prism last autosaved are lost.
stop_prism() {
  local console_user pids
  console_user=$(stat -f "%Su" /dev/console)
  if ! pids=$(prism_pids "$console_user"); then
    echo "Couldn't check whether Prism is running."
    return 1
  fi
  [[ -n "$pids" ]] || return 0
  echo "Stopping Prism..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..10}; do
    sleep 1
    if pids=$(prism_pids "$console_user") && [[ -z "$pids" ]]; then
      echo "Prism stopped."
      return 0
    fi
  done
  if [[ -n "$pids" ]]; then
    echo "Prism didn't stop; forcing it to quit."
    kill -KILL $pids 2>/dev/null
    sleep 1
    if pids=$(prism_pids "$console_user") && [[ -z "$pids" ]]; then
      echo "Prism stopped."
      return 0
    fi
  fi
  echo "Prism did not stop."
  return 1
}

remove_system_path() {
  local target="$1"
  local parent name rel top expected
  parent="$(dirname "$target")"
  name="$(basename "$target")"
  rel="${parent#/}"
  top="/${rel%%/*}"
  expected="$(cd -P "$top" 2>/dev/null && pwd -P)${parent#"$top"}"

  (
    cd -P "$parent" 2>/dev/null || exit 1
    if [[ "$(pwd -P)" != "$expected" ]]; then
      echo "Skipped $target: a folder in its path is a symlink."
      exit 0
    fi
    rm -rf "./$name"
  ) || echo "Failed to remove $target."
}

trash() {
  local logged_in_user="$1"
  local target_file="$2"
  local timestamp="$(date +%Y-%m-%d-%s)"
  local rand="$(jot -r 1 0 99999)"
  local trash=""

  # Paths in the user's home are moved to their Trash as the user: the user
  # controls those paths and ~/.Trash, so as root the move could be redirected
  # into a root-only directory. Other paths are deleted as root by
  # remove_system_path. At the login window there's no user Trash, so home
  # paths are skipped. Removal is best-effort and never fails the uninstall.
  if [[ "$target_file" == ~* ]]; then
    if [[ -z "$logged_in_user" || "$logged_in_user" == "root" || "$logged_in_user" == "loginwindow" ]]; then
      echo "Not logged into a non-root GUI; skipping $target_file."
      return
    fi
    # replace ~ with /Users/$logged_in_user
    target_file="/Users/$logged_in_user${target_file:1}"
    trash="/Users/$logged_in_user/.Trash"
  fi

  # If the target contains glob characters, expand it and remove each match.
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
      if [[ -z "$trash" ]]; then
        remove_system_path "$file"
      else
        # The per-match counter keeps matches that share a basename from
        # overwriting each other in the trash.
        sudo -u "$logged_in_user" mv -f "$file" "$trash/${file_name}_${timestamp}_${rand}_${i}" || echo "Failed to move $file to the Trash."
      fi
    done < <(compgen -G "$target_file" 2>/dev/null)
    if [[ "$matched" == false ]]; then
      echo "$target_file doesn't exist."
    fi
    return
  fi

  local file_name="$(basename "${target_file}")"

  if [[ -e "$target_file" ]]; then
    echo "removing $target_file."
    if [[ -z "$trash" ]]; then
      remove_system_path "$target_file"
    else
      sudo -u "$logged_in_user" mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}" || echo "Failed to move $target_file to the Trash."
    fi
  else
    echo "$target_file doesn't exist."
  fi
}

quit_application 'com.GraphPad.Prism' || stop_prism || exit 1
sudo rm -rf "$APPDIR/Prism 11.app"
sudo rm -rf '/Library/Application Support/GraphPad'
sudo rm -rf '/Library/GraphPad'
trash "$LOGGED_IN_USER" '~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.graphpad.prism.sfl*'
trash "$LOGGED_IN_USER" '~/Library/Application Support/GraphPad'
trash "$LOGGED_IN_USER" '~/Library/Caches/com.GraphPad.Prism'
trash "$LOGGED_IN_USER" '~/Library/HTTPStorages/com.GraphPad.Prism'
trash "$LOGGED_IN_USER" '~/Library/Logs/GraphPad'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.GraphPad.Prism.autocomplete.plist'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.GraphPad.Prism.plist'
trash "$LOGGED_IN_USER" '~/Library/Saved Application State/com.GraphPad.Prism.savedState'
trash "$LOGGED_IN_USER" '~/Library/WebKit/com.GraphPad.Prism'
