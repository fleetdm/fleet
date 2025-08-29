#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10
  if ! osascript -e "application id \"$bundle_id\" is running" >/dev/null 2>&1; then return; fi
  local console_user; console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then return; fi
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1 || true
    if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then break; fi
    sleep 1
  done
}

for id in "com.perforce.p4v" "com.perforce.p4merge" "com.perforce.p4admin"; do
  quit_application "$id"
done

rm -rf "/Applications/p4v.app" "/Applications/p4merge.app" "/Applications/p4admin.app" >/dev/null 2>&1 || true

for udir in /Users/* /var/root; do
  [[ -d "$udir/Library" ]] || continue
  rm -f  "$udir/Library/Preferences/com.perforce.p4v.plist" \
         "$udir/Library/Preferences/com.perforce.p4merge.plist" \
         "$udir/Library/Preferences/com.perforce.p4admin.plist" >/dev/null 2>&1 || true
  rm -rf "$udir/Library/Caches/com.perforce.p4v" \
         "$udir/Library/Caches/com.perforce.p4merge" \
         "$udir/Library/Caches/com.perforce.p4admin" >/dev/null 2>&1 || true
  rm -rf "$udir/Library/Application Support/Perforce" \
         "$udir/Library/Application Support/P4V" >/dev/null 2>&1 || true
done

echo "p4v suite uninstalled"
