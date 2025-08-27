#!/bin/sh
set -e
MOUNT_POINT="$(hdiutil attach -nobrowse -readonly "$INSTALLER_PATH" | awk '/\/Volumes\// {print $3; exit}')"
[ -n "$MOUNT_POINT" ] || { echo "failed to mount dmg"; exit 1; }
cleanup(){ hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true; }
trap cleanup EXIT

PKG="$(/usr/bin/find "$MOUNT_POINT" -maxdepth 2 -name '*.pkg' -print -quit)"
[ -n "$PKG" ] || { echo "omnissa pkg not found"; exit 1; }

sudo /usr/sbin/installer -pkg "$PKG" -target /
echo "omnissa horizon client installed"
