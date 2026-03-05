#!/bin/bash

# Safari Update Script
# This script runs softwareupdate to install Safari updates only

set -e

# Log file location
LOG_FILE="/var/log/safari_update.log"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log "Error: This script must be run as root (use sudo)"
    exit 1
fi

log "Starting Safari update process..."

# Run softwareupdate to install Safari updates only
# The --safari-only flag ensures only Safari updates are installed
if /usr/sbin/softwareupdate -i --safari-only; then
    log "Safari update completed successfully"
    exit 0
else
    log "Error: Safari update failed or no Safari updates available"
    exit 1
fi

