#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"

# find and install pkg file
PKG_FILE=$(find "$TMPDIR" -name "*.pkg" -type f | head -n 1)
if [ -z "$PKG_FILE" ]; then
  echo "Error: No .pkg file found in extracted archive"
  exit 1
fi
sudo installer -pkg "$PKG_FILE" -target /

