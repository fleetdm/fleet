#!/bin/sh
set -e
U="$(scutil <<< 'show State:/Users/ConsoleUser' | awk '/Name :/ {print $3}')"

quit_app(){ b="$1"; if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then cu="$(stat -f "%Su" /dev/console || true)"; [ "$cu" = "root" ] && return 0; osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true; sleep 2; fi; }
remove_label(){ l="$1"; launchctl remove "$l" >/dev/null 2>&1 || true; sudo launchctl remove "$l" >/dev/null 2>&1 || true; [ -n "$U" ] && launchctl bootout "gui/$(id -u "$U" 2>/dev/null)" "$l" >/dev/null 2>&1 || true; sudo launchctl bootout system "$l" >/dev/null 2>&1 || true; rm -f "/Library/LaunchAgents/$l.plist" "/Library/LaunchDaemons/$l.plist" >/dev/null 2>&1 || true; }
trash_user(){ user="$1"; t="$2"; case "$t" in "~/"*) t="/Users/$user/${t#~/}";; esac; [ -e "$t" ] || return 0; ts="$(date +%Y%m%d%H%M%S)"; r="$(od -An -N2 -i /dev/random 2>/dev/null | tr -d ' ')"; d="/Users/$user/.Trash/$(basename "$t")_${ts}_${r}"; mv -f "$t" "$d" 2>/dev/null || sudo -u "$user" mv -f "$t" "$d" 2>/dev/null || true; }
forget_pkg(){ pat="$1"; for id in $(pkgutil --pkgs | grep -E "$pat"); do sudo pkgutil --forget "$id" >/dev/null 2>&1 || true; done; }

for l in com.logi.cp-dev-mgr com.logi.optionsplus com.logi.optionsplus.updater com.logitech.LogiRightSight; do remove_label "$l"; done
for b in com.logi.cp-dev-mgr com.logi.optionsplus com.logi.optionsplus.driverhost com.logi.optionsplus.updater com.logitech.FirmwareUpdateTool com.logitech.logiaipromptbuilder; do quit_app "$b"; done

sudo rm -rf "/Applications/logioptionsplus.app" "/Applications/Utilities/Logi Options+ Driver Installer.bundle" "/Library/Application Support/Logitech.localized/LogiOptionsPlus" >/dev/null 2>&1 || true
rmdir "/Library/Application Support/Logitech.localized" >/dev/null 2>&1 || true

forget_pkg "^com\\.logitech\\.LogiRightSightForWebcams\\.pkg$"
forget_pkg "^com\\.logi\\."

[ -n "$U" ] && {
  trash_user "$U" "/Users/Shared/logi"
  trash_user "$U" "/Users/Shared/LogiOptionsPlus"
  trash_user "$U" "~/Library/Application Support/LogiOptionsPlus"
  trash_user "$U" "~/Library/Preferences/com.logi.cp-dev-mgr.plist"
  trash_user "$U" "~/Library/Preferences/com.logi.optionsplus.driverhost.plist"
  trash_user "$U" "~/Library/Preferences/com.logi.optionsplus.plist"
  trash_user "$U" "~/Library/Saved Application State/com.logi.optionsplus.savedState"
  trash_user "$U" "~/Library/Application Support/com.apple.sharedfilelist/com.apple.LSSharedFileList.ApplicationRecentDocuments/com.logi.optionsplus*.sfl"
}
echo "logitech options plus uninstalled"
