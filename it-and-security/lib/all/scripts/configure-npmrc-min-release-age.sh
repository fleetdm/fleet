#!/usr/bin/env bash
# Enforce min-release-age >= 0.5 in per-user ~/.npmrc on macOS and Linux.
# - Adds min-release-age=0.5 when the key is missing.
# - Replaces assignments strictly below 0.5 with min-release-age=0.5.
# - Leaves assignments at or above 0.5 and all other lines unchanged.

set -euo pipefail

MIN_VAL="0.5"

process_npmrc() {
  local npmrc="$1"
  local tmp homedir owner
  homedir="$(dirname "$npmrc")"

  # Refuse to follow symlinks: as root, writing/chowning a symlinked ~/.npmrc
  # would change ownership of whatever the link points to.
  if [[ -L "$npmrc" ]]; then
    printf 'skipping symlinked npmrc: %s\n' "$npmrc" >&2
    return 0
  fi

  # Stage every write in a fresh temp file inside $homedir
  tmp="$(mktemp "$homedir/.npmrc.fleet.XXXXXX")" || return 0

  if [[ ! -f "$npmrc" ]]; then
    printf '%s\n' "min-release-age=${MIN_VAL}" >"$tmp"
  else
    awk -v MIN="$MIN_VAL" '
      function trim(s) {
        sub(/^[ \t\r]+/, "", s)
        sub(/[ \t\r]+$/, "", s)
        return s
      }
      /^[ \t]*#/ {
        print
        next
      }
      {
        line = $0
        eq = index(line, "=")
        if (eq == 0) {
          print
          next
        }
        left = trim(substr(line, 1, eq - 1))
        right = substr(line, eq + 1)
        if (tolower(left) != "min-release-age") {
          print
          next
        }
        match(line, /^[ \t]*/)
        indent = substr(line, RSTART, RLENGTH)
        val = trim(right)
        sub(/#.*$/, "", val)
        val = trim(val)
        v = val + 0
        if (v >= MIN) {
          print line
          kept = 1
        } else {
          print indent "min-release-age=" MIN
          kept = 1
        }
        next
      }
      END {
        if (!kept) {
          print "min-release-age=" MIN
        }
      }
    ' "$npmrc" >"$tmp"

    if cmp -s "$npmrc" "$tmp"; then
      rm -f "$tmp"
      return 0
    fi
  fi

  chmod 600 "$tmp" 2>/dev/null || true
  if owner="$(stat -f '%u:%g' "$homedir" 2>/dev/null)" || owner="$(stat -c '%u:%g' "$homedir" 2>/dev/null)"; then
    # -h: act on the link, never follow it (defense in depth in case $tmp was
    # swapped between mktemp and now).
    chown -h "$owner" "$tmp" 2>/dev/null || true
  fi

  if ! mv -f "$tmp" "$npmrc" 2>/dev/null; then
    rm -f "$tmp"
    return 1
  fi

  # Detect a post-rename symlink swap. The data we wrote is safe at our inode,
  # but the directory entry could have been re-pointed, so refuse to claim
  # success.
  if [[ -L "$npmrc" ]]; then
    printf 'detected symlink swap on %s after write\n' "$npmrc" >&2
    return 1
  fi
}

# Pick the per-OS home-directory root.
case "$(uname -s)" in
  Darwin) bases=(/Users/*) ;;
  Linux)  bases=(/home/*)  ;;
  *)      bases=()         ;;
esac

for base in "${bases[@]}"; do
  [[ -d "$base" ]] || continue
  name="${base##*/}"
  case "$name" in
    Guest | Shared | .localized) continue ;;
  esac
  process_npmrc "${base}/.npmrc"
done

exit 0
