#!/bin/bash

# Removes QGIS-LTR.app and closes it if it's running. QGIS and per-user QGIS
# data (profiles, plugins, caches), which QGIS also uses, are left in place.

APP_PATH="/Applications/QGIS-LTR.app"

# QGIS and QGIS LTR share a bundle identifier, so match processes by path.
quit_app_at_path() {
  local app_path="$1"
  local pattern="^${app_path}/Contents/MacOS/"
  pgrep -f "$pattern" >/dev/null || return 0
  echo "Closing $app_path..."
  pkill -TERM -f "$pattern"
  for _ in {1..10}; do
    pgrep -f "$pattern" >/dev/null || return 0
    sleep 1
  done
  echo "$app_path did not close; forcing it to quit."
  pkill -KILL -f "$pattern"
}

quit_app_at_path "$APP_PATH"
sudo rm -rf "$APP_PATH"
