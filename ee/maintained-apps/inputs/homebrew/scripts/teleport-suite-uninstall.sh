#!/bin/bash

# variables
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

expand_pkgid_and_map() {
  local PKGID="$1"
  local FUNC="$2"
  if [[ "$PKGID" == *"*" ]]; then
    local prefix="${PKGID%\*}"
    # Convert (.*) to .* for regex matching to handle patterns like (.*).com.example.app
    local regex_pattern=$(echo "$prefix" | sed 's/(\.\*)/.*/g')
    echo "Expanding wildcard for PKGID: $PKGID (pattern: ^${regex_pattern})"
    for receipt in $(pkgutil --pkgs | grep -E "^${regex_pattern}"); do
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
  sudo pkgutil --only-files --files "$PKGID" | sed "s|^|${FULL_INSTALL_LOCATION}/|" | tr '\n' '\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf

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

# Root removes the client-tools cache from a pinned working directory after
# checking its kernel path (/bin/pwd, not the shell's string walk): the home's
# owner controls ~/.tsh, so root following a symlink they swapped in could
# otherwise delete any "bin" directory on the system.
remove_tsh_bin() { # remove_tsh_bin <home directory>
  local tsh_dir="$1/.tsh" real_home
  real_home=$(cd "$1" && /bin/pwd -P) || return 1
  (
    cd "$tsh_dir" || exit 1
    if [[ "$(/bin/pwd -P)" != "$real_home/.tsh" ]]; then
      echo "refusing to remove $tsh_dir/bin: $tsh_dir resolves to $(/bin/pwd -P)"
      exit 1
    fi
    rm -rf bin
  )
}

# Remove teleport-suite packages (handles wildcard patterns with (.*))
remove_pkg_files '(.*).com.gravitational.teleport.tctl'
forget_pkg '(.*).com.gravitational.teleport.tctl'
remove_pkg_files '(.*).com.gravitational.teleport.tsh'
forget_pkg '(.*).com.gravitational.teleport.tsh'
remove_pkg_files 'com.gravitational.teleport'
forget_pkg 'com.gravitational.teleport'

# Explicitly remove apps from /Applications (in case pkgutil removal didn't catch them)
sudo rm -rf '/Applications/tctl.app'
sudo rm -rf '/Applications/tsh.app'

# Remove binaries
sudo rm -rf '/usr/local/bin/fdpass-teleport'
sudo rm -rf '/usr/local/bin/tbot'
sudo rm -rf '/usr/local/bin/tctl'
sudo rm -rf '/usr/local/bin/teleport'
sudo rm -rf '/usr/local/bin/tsh'

# Remove auto-updater client-tools caches for all users, then the console user's data.
cache_failed=false
for home in /Users/*; do
  [[ -d "$home/.tsh/bin" ]] || continue
  echo "Removing Teleport client-tools cache '$home/.tsh/bin'."
  if ! remove_tsh_bin "$home"; then
    echo "Failed to remove '$home/.tsh/bin'."
    cache_failed=true
  fi
done
trash "$LOGGED_IN_USER" '~/.tsh'
[[ "$cache_failed" = false ]] || exit 1

