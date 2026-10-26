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


remove_launchctl_service() {
  local service="$1"
  local booleans=("true" "false")
  local plist_status
  local paths
  local should_sudo

  echo "Removing launchctl service ${service}"

  # A wildcard label can't be used with launchctl or as a plist name, so expand
  # it to the labels of currently loaded services that match the pattern.
  local services=("$service")
  if [[ "$service" == *"*"* ]]; then
    local regex
    # Escape regex metacharacters, turn '*' into '.*', and anchor the pattern so
    # it matches a full label rather than a substring.
    regex=$(printf '%s' "$service" | sed -e 's/[][(){}.^$+?|\\]/\\&/g' -e 's/\*/.*/g')
    regex="^${regex}$"
    services=()
    local id
    # Match every loaded job by label regardless of PID; launchctl list reports
    # loaded-but-not-running jobs with a "-" in the PID column.
    while read -r _ _ id; do
      [[ "$id" =~ $regex ]] && services+=("$id")
    done < <(launchctl list 2>/dev/null | tail -n +2)
    if [[ ${#services[@]} -eq 0 ]]; then
      echo "No loaded launchctl service matches ${service}"
      return
    fi
  fi

  local service_label
  for service_label in "${services[@]}"; do
    for should_sudo in "${booleans[@]}"; do
      plist_status=$(launchctl list "${service_label}" 2>/dev/null)

      if [[ $plist_status == \{* ]]; then
        if [[ $should_sudo == "true" ]]; then
          sudo launchctl remove "${service_label}"
        else
          launchctl remove "${service_label}"
        fi
        sleep 1
      fi

      paths=(
        "/Library/LaunchAgents/${service_label}.plist"
        "/Library/LaunchDaemons/${service_label}.plist"
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
  done
}

trash() {
  local logged_in_user="$1"
  local target_file="$2"
  local timestamp="$(date +%Y-%m-%d-%s)"
  local rand="$(jot -r 1 0 99999)"
  local trash=""

  # At the login window there's no user Trash to move files into, so user
  # paths are skipped and system paths are deleted outright. Removal is
  # best-effort and never fails the uninstall.
  if [[ -z "$logged_in_user" || "$logged_in_user" == "root" || "$logged_in_user" == "loginwindow" ]]; then
    if [[ "$target_file" == ~* ]]; then
      echo "Not logged into a non-root GUI; skipping $target_file."
      return
    fi
  else
    # replace ~ with /Users/$logged_in_user
    if [[ "$target_file" == ~* ]]; then
      target_file="/Users/$logged_in_user${target_file:1}"
    fi
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
        mv -f "$file" "$trash/${file_name}_${timestamp}_${rand}_${i}" || echo "Failed to move $file to the Trash."
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
      mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}" || echo "Failed to move $target_file to the Trash."
    fi
  else
    echo "$target_file doesn't exist."
  fi
}

remove_launchctl_service 'com.binarynights.ForkLiftHelper'
remove_launchctl_service 'com.binarynights.ForkLiftMini'
quit_application 'com.binarynights.ForkLift' || exit 1
quit_application 'com.binarynights.ForkLiftMini'
sudo rm -rf '/Library/PrivilegedHelperTools/com.binarynights.ForkLiftHelper'
sudo rm -rf "$APPDIR/ForkLift.app"
trash "$LOGGED_IN_USER" '~/Library/Application Support/ForkLift'
trash "$LOGGED_IN_USER" '~/Library/Caches/com.binarynights.ForkLift'
trash "$LOGGED_IN_USER" '~/Library/Cookies/com.binarynights.ForkLift.binarycookies'
trash "$LOGGED_IN_USER" '~/Library/HTTPStorages/com.binarynights.ForkLift'
trash "$LOGGED_IN_USER" '~/Library/Logs/ForkLift'
trash "$LOGGED_IN_USER" '~/Library/Logs/ForkLiftMini'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.binarynights.ForkLift.plist'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.binarynights.ForkLiftMini.plist'
trash "$LOGGED_IN_USER" '~/Library/Saved Application State/com.binarynights.ForkLift.savedState'
