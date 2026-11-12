#!/bin/bash

# Logi Tune installs a vendor uninstall script that unloads its launchd
# services, kills running processes, removes the RightSight daemon, and
# deletes the app bundle, support files, and pkg receipts. Prefer it when
# present (any install of Logi Tune 3.4+ has it).
VENDOR_UNINSTALLER="/Library/Application Support/logitune/uninstall_app.sh"

if [ -f "$VENDOR_UNINSTALLER" ]; then
  sh "$VENDOR_UNINSTALLER"
  exit $?
fi

# Fallback cleanup for installs missing the vendor uninstaller.

# stop and remove per-user launch agents
for agent in com.logitech.logitune.launcher com.logitech.logitune.agent; do
  for user in $(who | grep console | awk '{print $1}'); do
    user_id=$(id -u "$user")
    launchctl asuser "$user_id" launchctl unload "/Library/LaunchAgents/${agent}.plist" 2>/dev/null
  done
  rm -f "/Library/LaunchAgents/${agent}.plist"
done

# stop and remove launch daemons (incl. the bundled RightSight daemon)
for daemon in com.logitech.logitune.updater com.logitech.logitune.crashpad com.logitech.LogiRightSight; do
  launchctl unload "/Library/LaunchDaemons/${daemon}.plist" 2>/dev/null
  rm -f "/Library/LaunchDaemons/${daemon}.plist"
done

# kill any remaining processes
for process in LogiTune LogiTuneAgent LogiTuneUpdater LogiTuneCrashpadHandler; do
  pkill -9 -x "$process" 2>/dev/null
done

# remove the app bundle (current and legacy install paths) and support files
rm -rf "/Applications/Logi Tune.app"
rm -rf "/Applications/LogiTune.app"
rm -rf "/Applications/LogiTuneInstaller.app"
rm -rf "/Library/Application Support/logitune"
rm -rf "/Users/Shared/logitune"
rm -f "/Users/Shared/LogiTuneInstallerStarted.txt"

# forget pkg receipts
pkgutil --forget com.logitech.pkg.logitune 2>/dev/null
pkgutil --forget com.logitech.logitune.installer 2>/dev/null

exit 0
