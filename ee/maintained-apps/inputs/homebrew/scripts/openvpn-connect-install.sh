#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"

# install pkg files (this runs the postinstall script which moves the app and creates the symlink)
sudo installer -pkg "$TMPDIR/OpenVPN_Connect_3_8_1(5790)_arm64_Installer_signed.pkg" -target /

# Wait for the postinstall script to complete and create the symlink
# The postinstall script creates /Applications/OpenVPN Connect.app as a symlink
MAX_WAIT=30
COUNTER=0
while [ $COUNTER -lt $MAX_WAIT ]; do
  if [ -L "/Applications/OpenVPN Connect.app" ] || [ -d "/Applications/OpenVPN Connect/OpenVPN Connect.app" ]; then
    echo "OpenVPN Connect app found"
    break
  fi
  sleep 1
  COUNTER=$((COUNTER + 1))
done

if [ $COUNTER -ge $MAX_WAIT ]; then
  echo "Warning: Timed out waiting for OpenVPN Connect app installation"
fi
