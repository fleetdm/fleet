#!/bin/sh

quit_app() {
  b="$1"
  # try a friendly quit if a GUI user is active
  if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then
    cu="$(stat -f "%Su" /dev/console 2>/dev/null || true)"
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
  # hard stop fallback
  pkill -f "$b" >/dev/null 2>&1 || true
}

BUNDLE_ID="com.backblaze.bzbmenu"
quit_app "$BUNDLE_ID"

# Stop and remove launchctl services
for svc in com.backblaze.bzbmenu com.backblaze.bzserv; do
  launchctl bootout "system/$svc" >/dev/null 2>&1 || true
  launchctl bootout "gui/$(id -u)/$svc" >/dev/null 2>&1 || true
  launchctl remove "$svc" >/dev/null 2>&1 || true
done

# Remove app bundles and related files
rm -rf "/Applications/Backblaze.app" \
       "/Applications/BackblazeRestore.app" \
       "/Library/PreferencePanes/BackblazeBackup.prefPane" >/dev/null 2>&1 || true

# Remove diagnostic reports
rm -f /Library/Logs/DiagnosticReports/bzbmenu_*.*_resource.diag >/dev/null 2>&1 || true

# Remove package data and preferences
rm -rf "/Library/Backblaze.bzpkg" >/dev/null 2>&1 || true

# Per-user cleanup
for udir in /Users/* /var/root; do
  [ -d "$udir/Library" ] || continue
  rm -rf "$udir/Library/Logs/BackblazeGUIInstaller" >/dev/null 2>&1 || true
  rm -f  "$udir/Library/Preferences/com.backblaze.bzbmenu.plist" >/dev/null 2>&1 || true
done

echo "backblaze uninstalled"
