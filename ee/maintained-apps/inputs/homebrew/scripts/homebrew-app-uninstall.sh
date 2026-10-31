#!/bin/bash

# variables
APPDIR="/Applications/"
LOGGED_IN_USER=$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')
# functions

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

sudo rm -rf "$APPDIR/Homebrew.app" || exit $?

# Skip per-user cleanup when nobody is logged in to the GUI.
if [[ -n "$LOGGED_IN_USER" && "$LOGGED_IN_USER" != "root" && "$LOGGED_IN_USER" != "loginwindow" ]]; then
  trash "$LOGGED_IN_USER" '~/Library/Application Support/sh.brew.app'
  trash "$LOGGED_IN_USER" '~/Library/Caches/sh.brew.app'
  trash "$LOGGED_IN_USER" '~/Library/HTTPStorages/sh.brew.app'
  trash "$LOGGED_IN_USER" '~/Library/Preferences/sh.brew.app.plist'
fi
