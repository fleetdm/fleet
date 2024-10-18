#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"
# copy to the applications folder
sudo cp -R "$TMPDIR/Visual Studio Code.app" "$APPDIR"
