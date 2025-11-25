#!/bin/sh

# Mirror Homebrew's extraction logic
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"

# Fleet-specific: Dynamic architecture detection
# (Homebrew does this in Ruby, we do it in bash)
ARCH=$(uname -m)
PKG_FILE=$(find "$TMPDIR" -name "*_${ARCH}_Installer_signed.pkg" -type f | head -n 1)

if [ -z "$PKG_FILE" ]; then
  echo "Error: Could not find installer package for architecture: $ARCH"
  exit 1
fi

echo "Installing OpenVPN Connect for architecture: $ARCH"
echo "Package: $(basename "$PKG_FILE")"

# Mirror Homebrew's PKG installation
sudo installer -pkg "$PKG_FILE" -target /

# Wait for vendor's postinstall script to complete
sleep 5

# Fleet-specific: Restructure for osquery compatibility
# The vendor's PKG creates a symlink structure that osquery can't detect.
# We move components to standard locations while preserving functionality.

if [ -L "/Applications/OpenVPN Connect.app" ]; then
  echo "Restructuring installation for osquery compatibility..."

  # Remove symlink
  sudo rm "/Applications/OpenVPN Connect.app"

  # Move main app to standard location
  if [ -d "/Applications/OpenVPN Connect/OpenVPN Connect.app" ]; then
    sudo mv "/Applications/OpenVPN Connect/OpenVPN Connect.app" "/Applications/OpenVPN Connect.app"
    echo "Moved app to /Applications/OpenVPN Connect.app"
  fi

  # Move uninstaller to accessible location
  # Conservative approach: Keep vendor's uninstaller as safety net
  if [ -f "/Applications/OpenVPN Connect/Uninstall OpenVPN Connect.app" ]; then
    sudo mv "/Applications/OpenVPN Connect/Uninstall OpenVPN Connect.app" "/Applications/Uninstall OpenVPN Connect.app"
    echo "Moved uninstaller to /Applications/"
  fi

  # Remove now-empty directory
  if [ -d "/Applications/OpenVPN Connect" ]; then
    sudo rmdir "/Applications/OpenVPN Connect" 2>/dev/null && echo "Removed empty directory" || echo "Warning: Directory not empty, keeping it"
  fi
fi

# Verify installation
if [ -d "/Applications/OpenVPN Connect.app" ]; then
  echo "Installation complete: /Applications/OpenVPN Connect.app"
else
  echo "Error: App not found at expected location"
  exit 1
fi
