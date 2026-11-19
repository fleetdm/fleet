#!/bin/bash

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


# Browsers also run this executable as the extension's native messaging host,
# with the manifest path or a chrome-extension:// origin as its argument.
mozilla_vpn_pids() {
  ps -axo pid=,args= \
    | grep -E '^ *[0-9]+ /Applications/Mozilla VPN\.app/Contents/MacOS/Mozilla VPN( |$)' \
    | grep -vE 'mozillavpn\.json|chrome-extension:' \
    | awk '{print $1}'
}

# Mozilla VPN ignores the AppleScript quit. SIGTERM is its own shutdown path:
# it disconnects the VPN, then exits.
quit_mozilla_vpn() {
  local pids
  pids=$(mozilla_vpn_pids)
  [[ -n "$pids" ]] || return 0
  echo "Quitting Mozilla VPN..."
  kill -TERM $pids 2>/dev/null
  for _ in {1..20}; do
    sleep 1
    pids=$(mozilla_vpn_pids)
    if [[ -z "$pids" ]]; then
      echo "Mozilla VPN quit successfully."
      return 0
    fi
  done
  echo "Mozilla VPN didn't quit; forcing it to quit."
  kill -KILL $pids 2>/dev/null
  sleep 1
  if [[ -n "$(mozilla_vpn_pids)" ]]; then
    echo "Mozilla VPN did not quit."
    return 1
  fi
  echo "Mozilla VPN quit."
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

quit_mozilla_vpn || exit 1
remove_pkg_files 'org.mozilla.macos.FirefoxVPN'
forget_pkg 'org.mozilla.macos.FirefoxVPN'
trash "$LOGGED_IN_USER" '/Library/Application Support/Crash Reporter/Mozilla VPN_*'
trash "$LOGGED_IN_USER" '/Library/Logs/DiagnosticReports/Mozilla VPN_*'
trash "$LOGGED_IN_USER" '~/Library/Application Support/CrashReporter/Mozilla VPN_*'
trash "$LOGGED_IN_USER" '~/Library/Application Support/Mozilla/Mozilla VPN'
trash "$LOGGED_IN_USER" '~/Library/Application Support/mozillavpn.txt'
trash "$LOGGED_IN_USER" '~/Library/Caches/Mozilla VPN'
trash "$LOGGED_IN_USER" '~/Library/Saved Application State/org.mozilla.macos.FirefoxVPN.savedState'

