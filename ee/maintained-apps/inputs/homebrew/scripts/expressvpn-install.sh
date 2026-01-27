#!/bin/sh

set -e  # Exit on any error

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
echo "Extracting zip archive..."
unzip "$INSTALLER_PATH" -d "$TMPDIR"

# find and install pkg file
echo "Searching for .pkg file in extracted archive..."
PKG_FILE=$(find "$TMPDIR" -name "*.pkg" -type f | head -n 1)
if [ -z "$PKG_FILE" ]; then
  echo "Error: No .pkg file found in extracted archive"
  echo "Contents of $TMPDIR:"
  ls -la "$TMPDIR" || true
  exit 1
fi

echo "Found pkg file: $PKG_FILE"
echo "Installing package..."
if ! sudo installer -pkg "$PKG_FILE" -target /; then
  echo "Error: Package installation failed"
  exit 1
fi

echo "Package installed successfully"


