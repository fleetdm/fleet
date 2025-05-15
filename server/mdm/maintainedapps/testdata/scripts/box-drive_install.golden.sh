#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# install pkg files
sudo installer -pkg "$TMPDIR/BoxDrive-2.40.345.pkg" -target /
