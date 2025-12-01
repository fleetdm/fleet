#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"
# rename pkg file to match expected name (Homebrew behavior)
mv "$TMPDIR"/DisplayLinkManager-14.2*.pkg "$TMPDIR/DisplayLinkManager-14.2.pkg"
# install pkg files
sudo installer -pkg "$TMPDIR/DisplayLinkManager-14.2.pkg" -target /

