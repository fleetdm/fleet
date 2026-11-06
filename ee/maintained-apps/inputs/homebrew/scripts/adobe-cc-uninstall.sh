#!/bin/bash

quit_app() {
  b="$1"
  # try a friendly quit if a GUI user is active
  app_running=$(osascript -e "application id \"$b\" is running" 2>/dev/null)
  if [ "$app_running" = "true" ]; then
    cu="$(stat -f "%Su" /dev/console 2>/dev/null || true)"
    if [ "$(id -u)" -ne 0 ] || [ "$cu" != "root" ]; then
      i=0
      while [ "$i" -lt 10 ]; do
        osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true
        sleep 1
        if still_running=$(osascript -e "application id \"$b\" is running" 2>/dev/null) && [ "$still_running" = "false" ]; then break; fi
        i=$((i+1))
      done
    fi
  fi
}

# hard stop anything still running from an app bundle removed below
kill_app_at_path() {
  pattern="^$1/Contents/"
  pgrep -f "$pattern" >/dev/null 2>&1 || return 0
  pkill -TERM -f "$pattern" >/dev/null 2>&1 || true
  i=0
  while [ "$i" -lt 5 ]; do
    sleep 1
    pgrep -f "$pattern" >/dev/null 2>&1 || return 0
    i=$((i+1))
  done
  pkill -KILL -f "$pattern" >/dev/null 2>&1 || true
}

BUNDLE_ID="com.adobe.acc.AdobeCreativeCloud"
quit_app "$BUNDLE_ID"
kill_app_at_path "/Applications/Adobe Creative Cloud.app"
kill_app_at_path "/Applications/Creative Cloud.app"
kill_app_at_path "/Applications/Utilities/Adobe Creative Cloud/ACC/Creative Cloud.app"

# Try the official Adobe Creative Cloud Uninstaller.app first
UNINST_APP=""
for p in \
  "/Applications/Utilities/Adobe Creative Cloud/Utils/Creative Cloud Uninstaller.app" \
  "/Applications/Adobe Creative Cloud/Utils/Creative Cloud Uninstaller.app" \
  "/Applications/Utilities/Adobe Creative Cloud/Creative Cloud Uninstaller.app"
do
  [ -d "$p" ] && { UNINST_APP="$p"; break; }
done
[ -z "$UNINST_APP" ] && UNINST_APP="$(/usr/bin/find /Applications /Applications/Utilities -maxdepth 5 -type d -iname '*creative*cloud*uninstaller*.app' -print -quit 2>/dev/null)"

if [ -n "$UNINST_APP" ] && [ -d "$UNINST_APP" ]; then
  BIN="$(/usr/bin/find "$UNINST_APP/Contents/MacOS" -maxdepth 1 -type f -perm -111 -print -quit 2>/dev/null)"
  [ -n "$BIN" ] && { "$BIN" -uninstall --force >/dev/null 2>&1 || "$BIN" >/dev/null 2>&1 || true; }
fi

# Remove app bundles
rm -rf "/Applications/Adobe Creative Cloud.app" \
       "/Applications/Creative Cloud.app" \
       "/Applications/Utilities/Adobe Creative Cloud/ACC/Creative Cloud.app" >/dev/null 2>&1 || true

# System support files
rm -rf "/Library/Application Support/Adobe/Creative Cloud" \
       "/Library/Application Support/Adobe/Adobe Desktop Common" >/dev/null 2>&1 || true
rm -f  "/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" >/dev/null 2>&1 || true
rm -f  /var/db/receipts/com.adobe.acc*.bom /var/db/receipts/com.adobe.acc*.plist >/dev/null 2>&1 || true

# Per-user cleanup
for udir in /Users/* /var/root; do
  [ -d "$udir/Library" ] || continue
  rm -rf "$udir/Library/Application Support/Adobe/Creative Cloud" \
         "$udir/Library/Caches/com.adobe.acc.AdobeCreativeCloud" \
         "$udir/Library/Logs/Adobe/Creative Cloud" >/dev/null 2>&1 || true
  rm -f  "$udir/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" >/dev/null 2>&1 || true
done

echo "adobe creative cloud uninstalled"
