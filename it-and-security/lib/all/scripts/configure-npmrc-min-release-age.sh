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

  if [[ ! -f "$npmrc" ]]; then
    printf '%s\n' "min-release-age=${MIN_VAL}" >"$npmrc"
    if owner="$(stat -f '%u:%g' "$homedir" 2>/dev/null)" || owner="$(stat -c '%u:%g' "$homedir" 2>/dev/null)"; then
      chown "$owner" "$npmrc" 2>/dev/null || true
    fi
    chmod 644 "$npmrc" 2>/dev/null || true
    return 0
  fi

  tmp="$(mktemp "${TMPDIR:-/tmp}/fleet-npmrc.XXXXXX")"

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

  if ! cmp -s "$npmrc" "$tmp"; then
    cat "$tmp" >"$npmrc"
  fi
  rm -f "$tmp"
  if owner="$(stat -f '%u:%g' "$homedir" 2>/dev/null)" || owner="$(stat -c '%u:%g' "$homedir" 2>/dev/null)"; then
    chown "$owner" "$npmrc" 2>/dev/null || true
  fi
  chmod 644 "$npmrc" 2>/dev/null || true
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
