#!/bin/sh
set -e
APPDIR="/Applications"
U="$(scutil <<< 'show State:/Users/ConsoleUser' | awk '/Name :/ {print $3}')"

trash_user() {
  user="$1"; t="$2"
  [ -e "$t" ] || return 0
  ts="$(date +%Y%m%d%H%M%S)"
  r="$(od -An -N2 -i /dev/random 2>/dev/null | tr -d ' ')"
  d="/Users/$user/.Trash/$(basename "$t")_${ts}_${r}"
  mv -f "$t" "$d" 2>/dev/null || sudo -u "$user" mv -f "$t" "$d" 2>/dev/null || true
}

sudo rm -rf "$APPDIR/p4v.app" "$APPDIR/p4merge.app" "$APPDIR/p4admin.app" || true
[ -n "$U" ] && {
  trash_user "$U" "/Users/$U/Library/Preferences/com.perforce.p4v"
  trash_user "$U" "/Users/$U/Library/Preferences/com.perforce.p4v.plist"
  trash_user "$U" "/Users/$U/Library/Saved Application State/com.perforce.p4v.savedState"
}
echo "p4v uninstalled"
