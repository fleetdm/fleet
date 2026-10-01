#!/bin/bash
# Print the last N Fleet table migrations sorted by number, each labeled with the first compared ref that contains it.
#
# usage: migration-list.sh [-n count] [-r releases] [-C repo_dir] [-b base_ref] [-f]
#   -n  number of migrations to print (default 45)
#   -r  number of recent releases to compare (default 8)
#   -C  fleet repo dir (default: current dir)
#   -b  also compare HEAD, and mark migrations HEAD added or renamed since its merge base with base_ref
#   -f  run git fetch origin --tags first
#   -h  show this help
#
# Compares releases oldest first, then origin/main, then HEAD with -b. Releases are fleet-vX.Y.Z tags, or origin/rc-minor-fleet-vX.Y.Z /
# origin/rc-patch-fleet-vX.Y.Z branches for versions not tagged yet, labeled with their version. origin/main is labeled main, and HEAD is labeled with base_ref as "HEAD (target main)".
# Rows added or renamed in HEAD are marked with <-- NEW, and other rows whose label differs from the last row not marked NEW with <--

set -euo pipefail

count=45
release_count=8
repo_dir=.
base_ref=""
fetch=false
print_usage() {
  sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
}

while getopts "n:r:C:b:fh" opt; do
  case $opt in
    n) count=$OPTARG ;;
    r) release_count=$OPTARG ;;
    C) repo_dir=$OPTARG ;;
    b) base_ref=$OPTARG ;;
    f) fetch=true ;;
    h) print_usage; exit 0 ;;
    *) print_usage; exit 1 ;;
  esac
done

cd "$repo_dir"
migrations_dir=server/datastore/mysql/migrations/tables

if $fetch; then
  git fetch origin --tags -q
fi

work_dir=$(mktemp -d)
trap 'rm -r "$work_dir"' EXIT

# Write one "version ref" line per release, oldest first. A tag replaces the RC branch of the same version.
{
  # List release tags as "1 4.92.2 fleet-v4.92.2", the leading 1 sorts tags before RC branches of the same version
  git tag -l 'fleet-v*.*.*' | grep -E '^fleet-v[0-9]+\.[0-9]+\.[0-9]+$' | sed -E 's/^fleet-v(.*)$/\1 &/' | sed 's/^/1 /'
  # List RC branches as "2 4.93.0 origin/rc-minor-fleet-v4.93.0"
  git branch -r --format='%(refname:short)' | grep -E '^origin/rc-(minor|patch)-fleet-v[0-9]+\.[0-9]+\.[0-9]+$' | sed -E 's/^.*-fleet-v(.*)$/\1 &/' | sed 's/^/2 /'
# Sort by version then tag first, keep the first ref per version, print it as "version ref" and keep the newest releases
} | sort -k2,2V -k1,1n | awk '!seen[$2]++ { print $2 " " $3 }' | tail -n "$release_count" > "$work_dir/releases"

echo "main origin/main" >> "$work_dir/releases"
touch "$work_dir/new_numbers"
if [[ -n "$base_ref" ]]; then
  echo "HEAD HEAD" >> "$work_dir/releases"
  # Write the version IDs of migration files added or renamed in HEAD since its merge base with base_ref
  git diff --find-renames --diff-filter=AR --name-only "$base_ref...HEAD" -- "$migrations_dir/" \
    | grep -v '_test\.go$' | grep -oE '/[0-9]{14}_' | tr -d '/_' > "$work_dir/new_numbers" || true
fi

list_migration_numbers() {
  # Print the version ID of each non-test migration file in the given ref
  git ls-tree --name-only "$1" "$migrations_dir/" | grep -v '_test\.go$' | grep -oE '/[0-9]{14}_' | tr -d '/_' | sort -u
}

# Write a "number label" line for every migration in every compared ref, in the order the refs are listed
while read -r label ref; do
  list_migration_numbers "$ref" | sed "s/\$/ $label/"
done < "$work_dir/releases" > "$work_dir/numbers_with_version"

echo "Compared (first match wins):" >&2
sed 's/^/  /' "$work_dir/releases" >&2
echo >&2

# With -b, list the migrations in HEAD so numbers it renamed or removed don't show up, plus origin/main's when HEAD targets another branch such as an RC
if [[ -n "$base_ref" ]]; then
  list_migration_numbers HEAD > "$work_dir/listed_numbers"
  if [[ "$(git rev-parse "$base_ref")" != "$(git rev-parse origin/main)" ]]; then
    list_migration_numbers origin/main >> "$work_dir/listed_numbers"
  fi
  sort -u -o "$work_dir/listed_numbers" "$work_dir/listed_numbers"
else
  # List every number from every release
  cut -d' ' -f1 "$work_dir/numbers_with_version" | sort -u > "$work_dir/listed_numbers"
fi

echo "version id | earliest ref that includes this migration"
# Pair each listed number with the first release that has it, or a blank label if no compared ref has it, then keep the last count rows
awk 'FILENAME == ARGV[1] { if (!($1 in first)) first[$1] = $2; next } { print $1 " " (($1 in first) ? first[$1] : "") }' "$work_dir/numbers_with_version" "$work_dir/listed_numbers" \
  | sort -k1,1 | tail -n "$count" | awk -v target="${base_ref#origin/}" '
  FILENAME == ARGV[1] { new[$1] = 1; next }
  {
    # Show the branch HEAD merges into after HEAD, such as "HEAD (target main)"
    shown = ($2 == "HEAD" && target != "") ? "HEAD (target " target ")" : $2
    rows++; number[rows] = $1; label[rows] = shown
    # Pad to the longest release or branch label, so the longer HEAD label pushes only its own arrow out
    if ($2 != "HEAD" && length($2) > width) width = length($2)
  }
  END {
    # Pad labels to the longest one, mark rows in new_numbers with <-- NEW, and other rows whose label differs from the last row not in new_numbers with <--
    previous = ""
    for (i = 1; i <= rows; i++) {
      if (new[number[i]]) {
        printf "%s - %-*s  <-- NEW\n", number[i], width, label[i]
        continue
      }
      if (previous != "" && label[i] != previous) {
        printf "%s - %-*s  <--\n", number[i], width, label[i]
      } else {
        printf "%s - %s\n", number[i], label[i]
      }
      previous = label[i]
    }
  }
' "$work_dir/new_numbers" -
