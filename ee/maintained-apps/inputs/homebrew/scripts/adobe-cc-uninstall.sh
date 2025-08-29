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

quit_application "com.adobe.acc.AdobeCreativeCloud"

rm -rf "/Applications/Creative Cloud.app" \
       "/Applications/Adobe Creative Cloud.app" \
       "/Applications/Utilities/Adobe Creative Cloud/ACC/Creative Cloud.app" >/dev/null 2>&1 || true

rm -rf "/Library/Application Support/Adobe/Creative Cloud" \
       "/Library/Application Support/Adobe/Adobe Desktop Common" >/dev/null 2>&1 || true
rm -f  "/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" >/dev/null 2>&1 || true
rm -f  /var/db/receipts/com.adobe.acc*.{bom,plist} >/dev/null 2>&1 || true

for udir in /Users/* /var/root; do
  [[ -d "$udir/Library" ]] || continue
  rm -rf "$udir/Library/Application Support/Adobe/Creative Cloud" \
         "$udir/Library/Caches/com.adobe.acc.AdobeCreativeCloud" \
         "$udir/Library/Logs/Adobe/Creative Cloud" >/dev/null 2>&1 || true
  rm -f  "$udir/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" >/dev/null 2>&1 || true
done

echo "adobe creative cloud uninstalled"
