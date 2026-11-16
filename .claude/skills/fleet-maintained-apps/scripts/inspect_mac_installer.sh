#!/bin/bash
# Inspect a macOS installer (dmg, zip, or pkg) without installing anything: what
# the file really is, each top-level app's bundle id, versions, executable and
# signer, Gatekeeper's verdict, and pkg scripts that kill or relaunch the app.
#
# Usage: inspect_mac_installer.sh <https-url | local file> [--sha256 <expected>] [--keep] [--max-mb N]
#   --keep    leave the image mounted / payload extracted so you can run
#             check_queries.py --app-path against the bundles; prints how to clean up.
#   --max-mb  refuse downloads larger than this (default 2048).
# Downloads go to a fresh temp dir; nothing from the installer is executed.
set -uo pipefail

src=${1:?usage: inspect_mac_installer.sh <https-url | file> [--sha256 <expected>] [--keep] [--max-mb N]}
shift
want_sha="" keep=0 max_mb=2048
while [ $# -gt 0 ]; do
  case $1 in
    --sha256) want_sha=$(printf '%s' "$2" | tr '[:upper:]' '[:lower:]'); shift ;;
    --keep) keep=1 ;;
    --max-mb) max_mb=$2; shift ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

LSREGISTER=/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister
work=$(mktemp -d "${TMPDIR:-/tmp}/fma-inspect.XXXXXX")
# Resolve /var -> /private/var so the path matches what `mount` reports.
work=$(cd "$work" && pwd -P)
mnt="$work/mnt"
apps_seen=()

cleanup() {
  for a in "${apps_seen[@]+"${apps_seen[@]}"}"; do "$LSREGISTER" -u "$a" >/dev/null 2>&1; done
  if [ -d "$mnt" ]; then
    hdiutil detach "$mnt" >/dev/null 2>&1 || { sleep 2; hdiutil detach "$mnt" -force >/dev/null 2>&1; }
  fi
  if mount | grep -qF " on $mnt "; then
    echo "warning: $mnt is still mounted; detach it, then remove $work" >&2
  else
    rm -rf "$work"
  fi
}
# --keep drops this trap only after a full run, so early exits still clean up.
trap cleanup EXIT
trap 'exit 1' HUP INT TERM PIPE

if [[ $src == http://* ]]; then
  echo "refusing plain http: $src" >&2; exit 2
elif [[ $src == https://* ]]; then
  # Keep only the last response's headers (the redirects' come first).
  head=$(curl -sSIL --proto =https --max-time 60 "$src" | tr -d '\r' | awk '/^HTTP\//{b=""} {b=b $0 "\n"} END{printf "%s", b}') ||
    { echo "HEAD request failed" >&2; exit 1; }
  final=$(curl -sSIL --proto =https --max-time 60 -o /dev/null -w '%{url_effective}' "$src")
  len=$(printf '%s' "$head" | awk 'tolower($1)=="content-length:"{l=$2} END{print l}')
  echo "Final URL:     ${final%%\?*}"
  printf '%s\n' "$head" | tr -d '\r' | grep -iE '^(last-modified|content-length|content-disposition):' | sed 's/^/               /'
  if [ -n "$len" ] && [ "$len" -gt $((max_mb * 1024 * 1024)) ]; then
    echo "size $((len / 1024 / 1024)) MB exceeds --max-mb $max_mb; use a range request instead (references/identity-verification.md)" >&2
    exit 1
  fi
  # Fleet names the installer after Content-Disposition too. basename + the
  # character filter keep a hostile header from writing outside $work.
  name=$(printf '%s' "$head" | tr -d '\r' | grep -i '^content-disposition:' | tail -1 |
    sed -E "s/.*filename\*?=(UTF-8'')?\"?([^\";]+).*/\2/" | xargs -0 basename 2>/dev/null | tr -cd 'A-Za-z0-9._+ -')
  [ -n "$name" ] || name=$(basename "${final%%\?*}" | tr -cd 'A-Za-z0-9._+ -')
  file="$work/${name:-download}"
  # Before 8.4.0, --max-filesize doesn't stop a download whose size isn't announced.
  IFS=. read -r cmaj cmin _ <<<"$(curl --version | awk 'NR==1{print $2}')"
  if [ "${cmaj:-0}" -lt 8 ] || { [ "$cmaj" -eq 8 ] && [ "${cmin:-0}" -lt 4 ]; }; then
    echo "curl $cmaj.$cmin can't enforce --max-mb on a download without Content-Length; use curl 8.4.0 or later" >&2
    exit 1
  fi
  curl -sSL --proto =https --max-filesize $((max_mb * 1024 * 1024)) -o "$file" "$src" || { echo "download failed" >&2; exit 1; }
else
  file=$src
  [ -f "$file" ] || { echo "no such file: $file" >&2; exit 1; }
fi

sha=$(shasum -a 256 "$file" | awk '{print $1}')
echo "SHA-256:       $sha"
if [ -n "$want_sha" ]; then
  if [ "$sha" = "$want_sha" ]; then
    echo "               matches --sha256"
  else
    echo "               DOES NOT MATCH --sha256 $want_sha (rerun without --sha256 to inspect it anyway)"
    exit 1
  fi
fi

magic=$(head -c 4 "$file" | xxd -p)
if [ "$magic" = "78617221" ]; then kind=pkg
elif [ "$magic" = "504b0304" ]; then kind=zip
elif tail -c 512 "$file" | head -c 4 | grep -q koly; then kind=dmg
else kind=unknown
fi
echo "Real format:   $kind ($(file -b "$file" | cut -c1-60))  <- installer_format must be this"

show_app() {
  local app=$1 plist="$1/Contents/Info.plist" k
  apps_seen+=("$app")
  echo
  echo "App: ${app#"$work"/}"
  for k in CFBundleIdentifier CFBundleShortVersionString CFBundleVersion CFBundleExecutable LSMinimumSystemVersion; do
    printf '  %-28s %s\n' "$k" "$(/usr/libexec/PlistBuddy -c "Print :$k" "$plist" 2>/dev/null || echo '(missing)')"
  done
  printf '  %-28s %s\n' "Signer" "$(codesign -dv "$app" 2>&1 | sed -n 's/^TeamIdentifier=//p')  $(codesign -dvv "$app" 2>&1 | sed -n 's/^Authority=//p' | head -1)"
  printf '  %-28s %s\n' "Gatekeeper" "$(spctl -a -t execute -vv "$app" 2>&1 | tr '\n' ' ' | sed "s|$app: ||")"
  local nested
  nested=$(find "$app/Contents" -name '*.app' -maxdepth 3 -type d 2>/dev/null | head -5)
  [ -n "$nested" ] && printf '  %-28s %s\n' "Nested apps (open query?)" "$(echo "$nested" | sed "s|$app/||" | tr '\n' ' ')"
}

show_pkg() {
  local pkg=$1 out="$work/pkg.$RANDOM"
  echo
  echo "Package: ${pkg#"$work"/}"
  pkgutil --check-signature "$pkg" 2>/dev/null | sed -n 's/^ *1\. /  Signed by: /p' | head -1
  pkgutil --expand-full "$pkg" "$out" >/dev/null 2>&1 || { echo "  pkgutil --expand-full failed"; return; }
  grep -ho '<bundle [^>]*>' "$out"/PackageInfo "$out"/*/PackageInfo 2>/dev/null | grep -E 'CFBundleShortVersionString|id=' | head -6 | sed 's/^/  /'
  local s
  while IFS= read -r -d '' s; do
    echo "  script ${s#"$out"/}:"
    grep -nE 'pkill|killall|kill |osascript|open (-a|-b)|launchctl|rm -rf|/Applications/' "$s" 2>/dev/null | head -8 | sed 's/^/    /'
  done < <(find "$out" -path '*Scripts*' -type f \( -name 'pre*' -o -name 'post*' \) -print0 2>/dev/null)
  local app
  while IFS= read -r app; do show_app "$app"; done < <(find "$out" -path '*Payload*' -name '*.app' -type d -prune -maxdepth 6 2>/dev/null)
}

case $kind in
  dmg)
    mkdir -p "$mnt"
    # yes answers a license agreement; without it the attach is canceled. Check
    # hdiutil's own status: under pipefail, yes's SIGPIPE would fail the pipeline.
    yes | hdiutil attach -nobrowse -readonly -noautoopen -mountpoint "$mnt" "$file" >/dev/null
    [ "${PIPESTATUS[1]}" = 0 ] || { echo "hdiutil attach failed" >&2; exit 1; }
    echo "Top level:     $(ls "$mnt" | tr '\n' '|' | sed 's/|$//;s/|/ | /g')"
    while IFS= read -r app; do show_app "$app"; done < <(find "$mnt" -maxdepth 1 -name '*.app' -type d)
    while IFS= read -r d; do
      [ -n "$(find "$d" -maxdepth 1 -name '*.app' -type d)" ] && echo "Suite folder:  ${d#"$mnt"/} (custom scripts + lsregister; see macos-scripts.md)"
      while IFS= read -r app; do show_app "$app"; done < <(find "$d" -maxdepth 1 -name '*.app' -type d)
    done < <(find "$mnt" -mindepth 1 -maxdepth 1 -type d ! -name '*.app' ! -name '.*')
    while IFS= read -r p; do show_pkg "$p"; done < <(find "$mnt" -maxdepth 2 -name '*.pkg')
    [ -n "$(find "$mnt" -maxdepth 2 -name '*.zip')" ] && echo "Contains a zip: the generated script won't unpack it"
    ;;
  zip)
    listing=$(zipinfo -1 "$file")
    grep -qE '(^|/)\._|^__MACOSX/' <<<"$listing" && echo "AppleDouble entries: extract with ditto -x -k, never unzip"
    grep -iE '\.(dmg|pkg)$' <<<"$listing" | head -3 | sed 's/^/Nested installer: /'
    mkdir -p "$work/zip"
    ditto -x -k "$file" "$work/zip" || { echo "ditto failed" >&2; exit 1; }
    while IFS= read -r app; do show_app "$app"; done < <(find "$work/zip" -maxdepth 2 -name '*.app' -type d)
    while IFS= read -r p; do show_pkg "$p"; done < <(find "$work/zip" -maxdepth 2 -name '*.pkg')
    ;;
  pkg) show_pkg "$file" ;;
  *) echo "not a dmg, zip, or pkg: the FMA template can't install it" >&2 ;;
esac

if [ "$keep" = 1 ]; then
  trap - EXIT
  echo
  echo "Kept: $work"
  echo "Clean up: hdiutil detach \"$mnt\" 2>/dev/null; rm -rf \"$work\""
  for a in "${apps_seen[@]+"${apps_seen[@]}"}"; do echo "  (and run: $LSREGISTER -u \"$a\")"; done
fi
