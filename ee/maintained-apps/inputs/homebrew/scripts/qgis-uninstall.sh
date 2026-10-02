#!/bin/bash

# Removes every QGIS-final-<version>.app and closes QGIS if it's running. QGIS
# LTR and per-user QGIS data (profiles, plugins, caches), which QGIS LTR also
# uses, are left in place.

APPDIR="/Applications"

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

for app in "$APPDIR"/QGIS-final-*.app; do
  [[ -d "$app" ]] || continue
  quit_app_at_path "$app"
  echo "Removing $app"
  sudo rm -rf "$app"
done
