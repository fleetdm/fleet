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

fetch() { gh api "$repo/contents/$1" --jq .content 2>/dev/null | base64 -d; }

# Same filter as the ingester: version folders start with a digit and contain a dot.
# Others (e.g. Beta/, Portable/) are sibling packages or grouping folders.
versions=$(gh api "$repo/contents/$dir" --jq '.[] | select(.type == "dir") | .name | select(test("^[0-9].*[.]"))' | sort -V) || {
  echo "no folder at $dir: check the identifier's casing, or list manifests/$first/<Publisher>" >&2
  exit 1
}
if [ -z "$versions" ]; then
  echo "$dir has no version folders: the package was deleted or moved (see the commits below)" >&2
else
  echo "Newest versions in $dir:"
  printf '%s\n' "$versions" | tail -5 | sed 's/^/  /'
fi

echo "Recent commits to $dir:"
gh api "$repo/commits?path=$dir&per_page=6" \
  --jq '.[] | "  " + .commit.committer.date[0:10] + "  " + (.commit.message | split("\n")[0])'
[ -z "$versions" ] && exit 1

[ -n "$version" ] || version=$(printf '%s\n' "$versions" | tail -1)
echo
echo "=== $id $version installer manifest"
fetch "$dir/$version/$id.installer.yaml"

echo
locale=en-US
if ! fetch "$dir/$version/$id.locale.$locale.yaml" >/dev/null; then
  locale=$(fetch "$dir/$version/$id.yaml" | sed -n 's/^DefaultLocale: *//p' | tr -d '\r')
fi
echo "=== $id $version locale manifest ($locale)"
fetch "$dir/$version/$id.locale.$locale.yaml" | grep -E '^(PackageLocale|Publisher|PackageName|ShortDescription|Moniker):' || true

if [ "$diff" = 1 ]; then
  prev=$(printf '%s\n' "$versions" | sort -Vu | grep -B1 -xF "$version" | head -1)
  if [ -z "$prev" ] || [ "$prev" = "$version" ]; then
    echo "no earlier version to diff against" >&2
  else
    echo
    echo "=== installer manifest diff: $prev -> $version"
    diff -u <(fetch "$dir/$prev/$id.installer.yaml") <(fetch "$dir/$version/$id.installer.yaml") || true
  fi
fi
