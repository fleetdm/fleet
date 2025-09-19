#!/bin/sh

quit_and_kill() {
  b="$1"
  cu="$(stat -f "%Su" /dev/console 2>/dev/null || true)"

  # Friendly quit if a GUI user is logged in
  if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then
    if [ "$(id -u)" -ne 0 ] || [ "$cu" != "root" ]; then
      i=0
      while [ "$i" -lt 10 ]; do
        osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true
        if ! pgrep -f "$b" >/dev/null 2>&1; then break; fi
        i=$((i+1))
        sleep 1
      done
    fi
  fi

  # Hard kill by bundle id match and common names
  pkill -f "$b" >/dev/null 2>&1 || true
  pkill -f "[Ll]ogi Options\\+" >/dev/null 2>&1 || true
  pkill -f "logioptionsplus" >/dev/null 2>&1 || true
}

remove_label_everywhere() {
  l="$1"
  # system domain
  launchctl bootout system "$l" >/dev/null 2>&1 || true
  launchctl remove "$l" >/dev/null 2>&1 || true
  # per-user GUI domains
  if [ -d /Users ]; then
    for uhome in /Users/*; do
      [ -d "$uhome" ] || continue
      uid="$(id -u "$(basename "$uhome")" 2>/dev/null || true)"
      [ -n "$uid" ] || continue
      launchctl bootout "gui/$uid" "$l" >/dev/null 2>&1 || true
    done
  fi
  # remove plist files if present
  rm -f "/Library/LaunchAgents/$l.plist" "/Library/LaunchDaemons/$l.plist" >/dev/null 2>&1 || true
}

# 1) Stop processes
for b in \
  com.logi.optionsplus \
  com.logi.optionsplus.driverhost \
  com.logi.optionsplus.updater \
  com.logi.cp-dev-mgr \
  com.logitech.FirmwareUpdateTool \
  com.logitech.logiaipromptbuilder
do
  quit_and_kill "$b"
done

# 2) Unload agents/daemons everywhere so nothing relaunches
for l in \
  com.logi.optionsplus \
  com.logi.optionsplus.updater \
  com.logi.cp-dev-mgr \
  com.logitech.LogiRightSight
do
  remove_label_everywhere "$l"
done

# 3) Remove the app bundle (handle both names)
rm -rf "/Applications/Logi Options+.app" \
       "/Applications/logioptionsplus.app" \
       "/Applications/Utilities/Logi Options+ Driver Installer.bundle" \
       "/Applications/Utilities/LogiPluginService.app" \
       "/Applications/Logi Options Plus.app" >/dev/null 2>&1 || true

# 4) System support and receipts
rm -rf "/Library/Application Support/Logitech/LogiOptionsPlus" \
       "/Library/Application Support/Logitech.localized/LogiOptionsPlus" \
       "/Library/Application Support/Logi Options+" >/dev/null 2>&1 || true
rmdir  "/Library/Application Support/Logitech.localized" \
       "/Library/Application Support/Logi" >/dev/null 2>&1 || true

pkgutil --forget "com.logitech.LogiRightSightForWebcams.pkg" >/dev/null 2>&1 || true
for id in $(pkgutil --pkgs | grep -E '^com\.logi(\.|tech)' 2>/dev/null); do
  pkgutil --forget "$id" >/dev/null 2>&1 || true
done

# 5) Shared folders
rm -rf "/Users/Shared/logi" "/Users/Shared/LogiOptionsPlus" >/dev/null 2>&1 || true

# 6) Per-user cleanup
for uhome in /Users/*; do
  [ -d "$uhome/Library" ] || continue
  rm -rf "$uhome/Library/Application Support/LogiOptionsPlus" \
         "$uhome/Library/Saved Application State/com.logi.optionsplus.savedState" \
         "$uhome/Library/Caches/com.logi.optionsplus" >/dev/null 2>&1 || true
  rm -f  "$uhome/Library/Preferences/com.logi.cp-dev-mgr.plist" \
         "$uhome/Library/Preferences/com.logi.optionsplus.driverhost.plist" \
         "$uhome/Library/Preferences/com.logi.optionsplus.plist" >/dev/null 2>&1 || true
  rm -f  "$uhome/Library/Application Support/com.apple.sharedfilelist"/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.logi.optionsplus*.sfl* >/dev/null 2>&1 || true
done

echo "logi options+ uninstalled"
