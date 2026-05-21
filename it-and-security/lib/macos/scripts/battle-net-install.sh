#!/bin/sh

# Fleet downloads the .zip to $INSTALLER_PATH.
# The archive contains "Battle.net-Setup.app", which is the installer that
# downloads and installs the actual Battle.net.app into /Applications.

set -e

TMPDIR="$(mktemp -d /tmp/battle-net-install.XXXXXX)"
trap 'rm -rf "$TMPDIR"' EXIT

# Unzip the installer archive into a temp directory
/usr/bin/ditto -x -k "$INSTALLER_PATH" "$TMPDIR"

# Locate the setup .app inside the extracted contents
SETUP_APP="$(/usr/bin/find "$TMPDIR" -maxdepth 3 -type d -name '*Battle.net*Setup*.app' -print -quit)"

if [ -z "$SETUP_APP" ]; then
  echo "Could not locate Battle.net setup .app in extracted archive" >&2
  exit 1
fi

# Launch the setup binary directly so it runs to completion (non-interactive
# bootstrap; the Battle.net setup app handles the rest of the install).
SETUP_BIN="$SETUP_APP/Contents/MacOS/$(/usr/bin/defaults read "$SETUP_APP/Contents/Info" CFBundleExecutable)"

if [ ! -x "$SETUP_BIN" ]; then
  echo "Setup binary not executable: $SETUP_BIN" >&2
  exit 1
fi

"$SETUP_BIN" --lang=enUS --installpath="/Applications"
