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
  exit 1
fi

# Fix the symlink structure for osquery detection
# The vendor's postinstall creates:
#   /Applications/OpenVPN Connect/OpenVPN Connect.app (actual app)
#   /Applications/OpenVPN Connect.app (symlink)
# osquery doesn't follow symlinks, so we need to move the actual app to the expected location

echo "Restructuring app for osquery detection..."

# Remove the symlink
if [ -L "/Applications/OpenVPN Connect.app" ]; then
  sudo rm "/Applications/OpenVPN Connect.app"
  echo "Removed symlink"
fi

# Move the actual app to the standard location
if [ -d "/Applications/OpenVPN Connect/OpenVPN Connect.app" ]; then
  sudo mv "/Applications/OpenVPN Connect/OpenVPN Connect.app" "/Applications/OpenVPN Connect.app"
  echo "Moved app to /Applications/OpenVPN Connect.app"

  # Clean up empty directory
  sudo rmdir "/Applications/OpenVPN Connect" 2>/dev/null || echo "Directory not empty, keeping it"
fi

# Verify the app exists in the correct location
if [ ! -d "/Applications/OpenVPN Connect.app" ]; then
  echo "Error: App not found at /Applications/OpenVPN Connect.app"
  exit 1
fi

echo "Installation complete"
