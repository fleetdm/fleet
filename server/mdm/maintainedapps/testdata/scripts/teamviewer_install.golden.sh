#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# install pkg files
sudo installer -pkg "$TMPDIR/TeamViewer.pkg" -target /
