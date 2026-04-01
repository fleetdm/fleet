#!/bin/bash

# This script removes individual non-root user entries from /etc/sudoers and
# /etc/sudoers.d/ drop-in files. Group entries (%sudo, %wheel, etc.), aliases,
# Defaults directives, include directives, and comments are preserved.

set -e

BACKUP_SUFFIX=".bak.$(date +%s)"

# Patterns considered safe to keep (not individual non-root user grants):
#   - Empty/blank lines
#   - Comments (lines starting with #, but not legacy #include/#includedir)
#   - Defaults directives
#   - Group entries (lines starting with %)
#   - root entries
#   - Alias definitions (Host_Alias, User_Alias, Cmnd_Alias, Runas_Alias)
#   - Include directives (#include, #includedir, @include, @includedir)

is_safe_line() {
  local line="$1"

  # Empty or whitespace-only lines
  [[ -z "${line// /}" ]] && return 0

  # Comments (but not #include/#includedir which are handled below)
  if [[ "$line" =~ ^[[:space:]]*# ]]; then
    # Allow #include and #includedir directives
    [[ "$line" =~ ^[[:space:]]*#include(dir)?[[:space:]] ]] && return 0
    # Regular comment
    return 0
  fi

  # @include / @includedir directives
  [[ "$line" =~ ^[[:space:]]*@include(dir)?[[:space:]] ]] && return 0

  # Defaults directives
  [[ "$line" =~ ^[[:space:]]*Defaults ]] && return 0

  # Group entries (start with %)
  [[ "$line" =~ ^[[:space:]]*% ]] && return 0

  # root entries
  [[ "$line" =~ ^[[:space:]]*root[[:space:]] ]] && return 0

  # Alias definitions
  [[ "$line" =~ ^[[:space:]]*(Host_Alias|User_Alias|Cmnd_Alias|Runas_Alias)[[:space:]] ]] && return 0

  return 1
}

clean_sudoers_file() {
  local file="$1"

  if [ ! -f "$file" ]; then
    return
  fi

  local tmpfile
  tmpfile=$(mktemp)
  local changed=false

  while IFS= read -r line || [[ -n "$line" ]]; do
    if is_safe_line "$line"; then
      echo "$line" >> "$tmpfile"
    else
      echo "Removing non-root user entry from $file: $line"
      changed=true
    fi
  done < "$file"

  if [ "$changed" = true ]; then
    # Back up the original file before modifying
    cp "$file" "${file}${BACKUP_SUFFIX}"

    # Validate the new file with visudo before applying
    if visudo -c -f "$tmpfile" > /dev/null 2>&1; then
      cp "$tmpfile" "$file"
      chmod 0440 "$file"
      echo "Updated $file (backup saved as ${file}${BACKUP_SUFFIX})"
    else
      echo "ERROR: Modified $file failed visudo syntax check. Skipping this file."
      rm -f "${file}${BACKUP_SUFFIX}"
    fi
  fi

  rm -f "$tmpfile"
}

# Require root
if [ "$(id -u)" -ne 0 ]; then
  echo "ERROR: This script must be run as root."
  exit 1
fi

echo "Checking /etc/sudoers and /etc/sudoers.d/ for non-root user entries..."

# Clean the main sudoers file
clean_sudoers_file /etc/sudoers

# Clean all drop-in files in /etc/sudoers.d/
if [ -d /etc/sudoers.d ]; then
  for file in /etc/sudoers.d/*; do
    # Skip files ending in ~ or containing . (per sudoers.d convention)
    [[ "$file" == *~ ]] && continue
    [[ "$(basename "$file")" == *.* ]] && continue
    clean_sudoers_file "$file"
  done
fi

echo "Done. Non-root individual user entries have been removed from sudoers files."
