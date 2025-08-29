#!/bin/bash

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

quit_application "com.logi.optionsplus"

rm -rf "/Applications/Logi Options+.app" >/dev/null 2>&1 || true

rm -rf "/Library/Application Support/Logitech/LogiOptionsPlus" \
       "/Library/Application Support/Logi Options+" >/dev/null 2>&1 || true
rm -f  "/Library/Preferences/com.logi.optionsplus.plist" >/dev/null 2>&1 || true
rm -f  /var/db/receipts/com.logi.optionsplus*.{bom,plist} >/dev/null 2>&1 || true

for udir in /Users/* /var/root; do
  [[ -d "$udir/Library" ]] || continue
  rm -rf "$udir/Library/Application Support/Logi Options+" \
         "$udir/Library/Application Support/Logitech/LogiOptionsPlus" \
         "$udir/Library/Caches/com.logi.optionsplus" \
         "$udir/Library/Logs/Logi Options+" >/dev/null 2>&1 || true
  rm -f  "$udir/Library/Preferences/com.logi.optionsplus.plist" >/dev/null 2>&1 || true
done

echo "logi options+ uninstalled"
