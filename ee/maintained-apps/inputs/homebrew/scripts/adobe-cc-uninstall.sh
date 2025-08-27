#!/bin/sh
set -e
U="$(scutil <<< 'show State:/Users/ConsoleUser' | awk '/Name :/ {print $3}')"

bootout(){ l="$1"; launchctl remove "$l" >/dev/null 2>&1 || true; sudo launchctl remove "$l" >/dev/null 2>&1 || true; [ -n "$U" ] && launchctl bootout "gui/$(id -u "$U" 2>/dev/null)" "$l" >/dev/null 2>&1 || true; sudo launchctl bootout system "$l" >/dev/null 2>&1 || true; }
quit_app(){ b="$1"; if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then cu="$(stat -f "%Su" /dev/console || true)"; [ "$cu" = "root" ] && return 0; osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true; sleep 2; fi; }

UNINST="/Applications/Utilities/Adobe Creative Cloud/Utils/Creative Cloud Uninstaller.app/Contents/MacOS/Creative Cloud Uninstaller"
if [ -x "$UNINST" ]; then
  "$UNINST" -u >/dev/null 2>&1 && sudo "$UNINST" -u || "$UNINST" --force >/dev/null 2>&1 && sudo "$UNINST" --force || sudo "$UNINST"
fi

pluginkit -r "/Applications/Utilities/Adobe Sync/CoreSync/Core Sync.app/Contents/PlugIns/ACCFinderSync.appex" >/dev/null 2>&1 || true

for l in Adobe_Genuine_Software_Integrity_Service com.adobe.AdobeCreativeCloud com.adobe.acc.installer com.adobe.acc.installer.v2 com.adobe.ccxprocess; do bootout "$l"; done

quit_app "com.adobe.acc.AdobeCreativeCloud"
osascript -e 'tell application "System Events" to delete login item "Adobe Creative Cloud"' >/dev/null 2>&1 || true
pkill -f "Adobe Desktop Service|AdobeIPCBroker|AdobeCRDaemon" >/dev/null 2>&1 || true

sudo rm -rf "/Applications/Adobe Creative Cloud" \
            "/Applications/Utilities/Adobe Application Manager" \
            "/Applications/Utilities/Adobe Creative Cloud"* \
            "/Applications/Utilities/Adobe Installers/.Uninstall"* \
            "/Applications/Utilities/Adobe Sync" \
            "/Library/Internet Plug-Ins/AdobeAAMDetect.plugin" \
            "/Library/Logs/CreativeCloud" >/dev/null 2>&1 || true

sudo rm -rf "/Library/Application Support/Adobe/Adobe Desktop Common" \
            "/Library/Application Support/Adobe/AdobeApplicationManager" \
            "/Library/Application Support/Adobe/AdobeGC"* \
            "/Library/Application Support/Adobe/Creative Cloud Libraries" \
            "/Library/Application Support/Adobe/Extension Manager CC" \
            "/Library/Application Support/Adobe/OOBE" \
            "/Library/Application Support/Adobe/CEP/extensions/CC_*" \
            "/Library/Application Support/Adobe/CEP/extensions/com.adobe.ccx.*" \
            "/Library/Application Support/regid.*.com.adobe" >/dev/null 2>&1 || true

if [ -n "$U" ]; then
  H="/Users/$U"
  rm -rf "$H/Creative Cloud Files" \
         "$H/Library/Application Support/Adobe/OOBE" \
         "$H/Library/Application Support/Adobe/ExtensibilityLibrary" \
         "$H/Library/Logs/CreativeCloud" \
         "$H/Library/Logs/AdobeDownload.log" \
         "$H/Library/Logs/AdobeIPCBroker"* \
         "$H/Library/Logs/CoreSyncInstall.log" \
         "$H/Library/LaunchAgents/com.adobe.ccxprocess.plist" \
         "$H/Library/Application Scripts/com.adobe.accmac.ACCFinderSync" \
         "$H/Library/*/com.adobe.acc"* >/dev/null 2>&1 || true
fi
echo "adobe creative cloud uninstalled"
