#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"
# install pkg files
sudo installer -pkg "$TMPDIR/AcroRdrDC_2400221005_MUI.pkg" -target /
