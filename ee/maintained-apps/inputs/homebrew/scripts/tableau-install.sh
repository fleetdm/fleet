#!/bin/sh

# variables
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# Mount the DMG
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"

# Find the .pkg file in the mounted DMG
PKG_FILE=$(find "$MOUNT_POINT" -maxdepth 2 -name "*.pkg" -print -quit)

if [ -z "$PKG_FILE" ]; then
  echo "Error: No .pkg file found in DMG"
  hdiutil detach "$MOUNT_POINT"
  exit 1
fi

echo "Found pkg file: $PKG_FILE"

# Install the pkg
sudo installer -pkg "$PKG_FILE" -target /

# Detach the DMG
hdiutil detach "$MOUNT_POINT"
