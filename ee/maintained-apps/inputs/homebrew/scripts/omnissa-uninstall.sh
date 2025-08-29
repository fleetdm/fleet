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

BUNDLE_ID="com.omnissa.horizon.client.mac"
quit_application "$BUNDLE_ID"

rm -rf "/Applications/Omnissa Horizon Client.app" \
       "/Applications/VMware Horizon Client.app" >/dev/null 2>&1 || true

if [[ -f "/Library/LaunchDaemons/com.omnissa.horizon.CDSHelper.plist" ]]; then
  launchctl bootout system "/Library/LaunchDaemons/com.omnissa.horizon.CDSHelper.plist" >/dev/null 2>&1 || \
  launchctl unload -w "/Library/LaunchDaemons/com.omnissa.horizon.CDSHelper.plist" >/dev/null 2>&1 || true
fi
rm -f "/Library/LaunchDaemons/com.omnissa.horizon.CDSHelper.plist" >/dev/null 2>&1 || true
rm -f "/Library/PrivilegedHelperTools/com.omnissa.horizon.CDSHelper" >/dev/null 2>&1 || true

rm -rf "/Library/Application Support/WorkspaceONE/helper" >/dev/null 2>&1 || true
rm -rf "/Library/Application Support/Omnissa" >/dev/null 2>&1 || true
rm -f  "/Library/Preferences/${BUNDLE_ID}.plist" >/dev/null 2>&1 || true
rm -f  /var/db/receipts/com.omnissa.horizon*.{bom,plist} >/dev/null 2>&1 || true
rm -f  /var/db/receipts/com.vmware.horizon*.{bom,plist} >/dev/null 2>&1 || true

for udir in /Users/* /var/root; do
  [[ -d "$udir/Library" ]] || continue
  rm -rf "$udir/Library/Application Support/VMware Horizon Client" \
         "$udir/Library/Application Support/Omnissa Horizon Client" \
         "$udir/Library/Caches/${BUNDLE_ID}" \
         "$udir/Library/Logs/VMware Horizon Client" \
         "$udir/Library/Logs/Omnissa Horizon Client" >/dev/null 2>&1 || true
  rm -f  "$udir/Library/Preferences/${BUNDLE_ID}.plist" >/dev/null 2>&1 || true
done

echo "omnissa horizon client uninstalled"
