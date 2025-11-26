#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"

# discover the installer app by finding any .app that contains the installer binary
INSTALLER_APP=""
for app in "$TMPDIR"/*.app; do
  if [ -d "$app" ] && [ -f "$app/Contents/MacOS/logioptionsplus_installer" ]; then
    INSTALLER_APP="$app"
    break
  fi
done

# run the installer if found
if [ -n "$INSTALLER_APP" ] && [ -d "$INSTALLER_APP" ]; then
  "$INSTALLER_APP/Contents/MacOS/logioptionsplus_installer" --quiet
  EXIT_CODE=$?
  if [ $EXIT_CODE -ne 0 ]; then
    echo "Error: Installer exited with code $EXIT_CODE"
    exit $EXIT_CODE
  fi
  # cleanup: remove the installer app after successful installation
  rm -rf "$INSTALLER_APP"
else
  echo "Error: Installer app with logioptionsplus_installer binary not found in $TMPDIR"
  exit 1
fi

