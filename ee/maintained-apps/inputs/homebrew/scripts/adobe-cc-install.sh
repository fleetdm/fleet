#!/bin/sh
set -e
MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\// {print $3; exit}')"
[ -n "$MOUNT_POINT" ] || { echo "failed to mount dmg"; exit 1; }
cleanup(){ hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true; }
trap cleanup EXIT

BIN="$(/usr/bin/find "$MOUNT_POINT" -type f -path '*/Install.app/Contents/MacOS/Install' -print -quit)"
[ -z "$BIN" ] && BIN="$(/usr/bin/find "$MOUNT_POINT" -type f -path '*/Creative Cloud Installer.app/Contents/MacOS/Install' -print -quit)"
[ -n "$BIN" ] || { echo "adobe cc installer binary not found"; exit 1; }

if "$BIN" --mode=silent >/dev/null 2>&1; then
  sudo "$BIN" --mode=silent
elif "$BIN" --silent >/dev/null 2>&1; then
  sudo "$BIN" --silent
else
  sudo "$BIN"
fi
echo "adobe creative cloud installed"
