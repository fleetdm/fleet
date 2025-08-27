#!/bin/sh
set -e
quit_app(){ b="$1"; if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then cu="$(stat -f "%Su" /dev/console || true)"; [ "$cu" = "root" ] && return 0; osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true; sleep 2; fi; }
forget_pkg_like(){ pat="$1"; for id in $(pkgutil --pkgs | grep -E "$pat"); do sudo pkgutil --forget "$id" >/dev/null 2>&1 || true; done; }

APP1="/Applications/VMware Horizon Client.app"
APP2="/Applications/Omnissa Horizon Client.app"

quit_app "com.omnissa.horizon.client.mac"
quit_app "com.vmware.horizon"

sudo rm -rf "$APP1" "$APP2" >/dev/null 2>&1 || true
forget_pkg_like "(vmware|omnissa).*horizon"
rm -f "/Library/Preferences/com.vmware.horizon.plist" >/dev/null 2>&1 || true
echo "omnissa horizon client uninstalled"
