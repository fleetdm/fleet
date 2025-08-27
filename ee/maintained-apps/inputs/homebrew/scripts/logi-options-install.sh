#!/bin/sh
set -e
TMPDIR="$(mktemp -d /tmp/logiopts_XXXXXX)"
cleanup(){ rm -rf "$TMPDIR" >/dev/null 2>&1 || true; }
trap cleanup EXIT

unzip -q "$INSTALLER_PATH" -d "$TMPDIR"

INNER_BIN="$(/usr/bin/find "$TMPDIR" -type f -path '*/Logi*Installer*.app/Contents/MacOS/*' -print -quit)"
[ -z "$INNER_BIN" ] && INNER_BIN="$(/usr/bin/find "$TMPDIR" -type f -path '*/Options+*.app/Contents/MacOS/*' -print -quit)"
[ -z "$INNER_BIN" ] && INNER_BIN="$(/usr/bin/find "$TMPDIR" -type f -path '*/Install*.app/Contents/MacOS/*' -print -quit)"
[ -n "$INNER_BIN" ] || { echo "inner installer not found"; exit 1; }

if "$INNER_BIN" --quiet >/dev/null 2>&1; then
  sudo "$INNER_BIN" --quiet
else
  sudo "$INNER_BIN"
fi
echo "logitech options plus installed"
