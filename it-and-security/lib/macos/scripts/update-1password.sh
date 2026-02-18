#!/bin/bash

# 1Password Update Script
# This script downloads and installs the latest version of 1Password

set -e

# Log file location
LOG_FILE="/var/log/1password_update.log"

# 1Password download URL
DOWNLOAD_URL="https://downloads.1password.com/mac/1Password.pkg"

# Temporary file for the downloaded package
TMP_PKG="/tmp/1Password.pkg"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log "Error: This script must be run as root (use sudo)"
    exit 1
fi

log "Starting 1Password update process..."

# Download the latest 1Password package
log "Downloading 1Password from $DOWNLOAD_URL..."
if ! curl -fsSL -o "$TMP_PKG" "$DOWNLOAD_URL"; then
    log "Error: Failed to download 1Password package"
    exit 1
fi

log "Download complete. Installing 1Password..."

# Install the package
if installer -pkg "$TMP_PKG" -target /; then
    log "1Password installed/updated successfully"
else
    log "Error: 1Password installation failed"
    rm -f "$TMP_PKG"
    exit 1
fi

# Clean up the temporary package
rm -f "$TMP_PKG"
log "Cleanup complete. 1Password update finished successfully."
exit 0
