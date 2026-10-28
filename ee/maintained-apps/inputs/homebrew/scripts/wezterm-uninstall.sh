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
  local quit_success=false still_running
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1
    sleep 1
    if still_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
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
  local trash=""

  # Paths in the user's home are moved to their Trash as the user: the user
  # controls those paths and ~/.Trash, so as root the move could be redirected
  # into a root-only directory. Other paths are deleted as root. At the login
  # window there's no user Trash, so home paths are skipped. Removal is
  # best-effort and never fails the uninstall.
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
        rm -rf "$file" || echo "Failed to remove $file."
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
      rm -rf "$target_file" || echo "Failed to remove $target_file."
    else
      sudo -u "$logged_in_user" mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}" || echo "Failed to move $target_file to the Trash."
    fi
  else
    echo "$target_file doesn't exist."
  fi
}

quit_application 'com.github.wez.wezterm' || exit 1
sudo rm -rf "$APPDIR/WezTerm.app"
trash "$LOGGED_IN_USER" '~/.local/share/wezterm'
trash "$LOGGED_IN_USER" '~/Library/Saved Application State/com.github.wez.wezterm.savedState'
