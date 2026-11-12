#!/bin/bash
# Print a winget package's installer and locale manifests, its newest versions,
# and the recent commits to its folder (in-place edits, "Automatic deletion",
# "Moved:"). With --diff, also diff the installer manifest against the previous
# version's, which is how to find the field behind "failed to find installer".
#
# Usage: winget_manifest.sh <PackageIdentifier> [version] [--diff]
# Needs gh (authenticated). GitHub code search can't see winget-pkgs; this reads
# the contents API instead.
set -euo pipefail

id=${1:?usage: winget_manifest.sh <PackageIdentifier> [version] [--diff]}
shift
version="" diff=0
for arg in "$@"; do
  case $arg in
    --diff) diff=1 ;;
    *) version=$arg ;;
  esac
done

first=$(printf '%s' "${id:0:1}" | tr '[:upper:]' '[:lower:]')
dir="manifests/$first/${id//.//}"
repo=repos/microsoft/winget-pkgs

fetch() {
  local out
  out=$(gh api "$repo/contents/$1" --jq .content 2>/dev/null | base64 -d 2>/dev/null) && [ -n "$out" ] ||
    { echo "failed to fetch $1" >&2; return 1; }
  printf '%s\n' "$out"
}

# Same filter as the ingester: version folders start with a digit and contain a dot.
# Others (e.g. Beta/, Portable/) are sibling packages or grouping folders.
# A deleted or moved package's folder 404s, but its history still explains why.
if ! versions=$(gh api "$repo/contents/$dir" --jq '.[] | select(.type == "dir") | .name | select(test("^[0-9].*[.]"))' 2>/dev/null | sort -V); then
  echo "no folder at $dir: the package was deleted or moved (see the commits below), or the identifier's casing is wrong (list manifests/$first/<Publisher>)" >&2
  versions=""
elif [ -z "$versions" ]; then
  echo "$dir has no version folders: the package was deleted or moved (see the commits below)" >&2
else
  echo "Newest versions in $dir:"
  printf '%s\n' "$versions" | tail -5 | sed 's/^/  /'
fi

echo "Recent commits to $dir:"
commits=$(gh api "$repo/commits?path=$dir&per_page=6" \
  --jq '.[] | "  " + .commit.committer.date[0:10] + "  " + (.commit.message | split("\n")[0])')
printf '%s\n' "${commits:-  (none: nothing has ever lived at this path)}"
[ -z "$versions" ] && exit 1

[ -n "$version" ] || version=$(printf '%s\n' "$versions" | tail -1)
echo
echo "=== $id $version installer manifest"
fetch "$dir/$version/$id.installer.yaml"

echo
locale=en-US
if ! locale_yaml=$(fetch "$dir/$version/$id.locale.$locale.yaml" 2>/dev/null); then
  version_yaml=$(fetch "$dir/$version/$id.yaml")
  locale=$(printf '%s\n' "$version_yaml" | sed -n 's/^DefaultLocale: *//p' | tr -d '\r')
  locale_yaml=$(fetch "$dir/$version/$id.locale.$locale.yaml")
fi
echo "=== $id $version locale manifest ($locale)"
grep -E '^(PackageLocale|Publisher|PackageName|ShortDescription|Moniker):' <<<"$locale_yaml" || true

if [ "$diff" = 1 ]; then
  prev=$(printf '%s\n' "$versions" | sort -Vu | grep -B1 -xF "$version" | head -1)
  if [ -z "$prev" ] || [ "$prev" = "$version" ]; then
    echo "no earlier version to diff against" >&2
  else
    old=$(fetch "$dir/$prev/$id.installer.yaml")
    new=$(fetch "$dir/$version/$id.installer.yaml")
    echo
    echo "=== installer manifest diff: $prev -> $version"
    # diff exits 1 when the manifests differ, which is the expected case.
    diff -u <(printf '%s\n' "$old") <(printf '%s\n' "$new") || [ $? -eq 1 ]
  fi
fi
