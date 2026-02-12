#!/bin/bash

# Clawbot/moltbot uninstall script for macOS
# Removes Clawbot (also known as moltbot) and cleans up related files

set -e

echo "Starting Clawbot/moltbot uninstallation..."

# Function to log messages with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Ensure running as root
if [[ $EUID -ne 0 ]]; then
    log "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Kill any running Clawbot/moltbot processes
log "Stopping Clawbot/moltbot processes..."
pkill -f "[Cc]lawbot" || true
pkill -f "[Mm]oltbot" || true
sleep 2

# Unload and remove launch daemons/agents
for label in com.clawbot.agent com.moltbot.agent; do
    if launchctl list "$label" &>/dev/null; then
        log "Unloading launch daemon: $label"
        launchctl bootout system/"$label" 2>/dev/null || launchctl remove "$label" 2>/dev/null || true
    fi
done

for plist in /Library/LaunchDaemons/*clawbot* /Library/LaunchDaemons/*moltbot* \
             /Library/LaunchAgents/*clawbot* /Library/LaunchAgents/*moltbot*; do
    if [[ -f "$plist" ]]; then
        log "Removing launch plist: $plist"
        rm -f "$plist"
    fi
done

# Remove application bundles
for app in /Applications/Clawbot.app /Applications/Moltbot.app; do
    if [[ -d "$app" ]]; then
        log "Removing application: $app"
        rm -rf "$app"
    fi
done

# Remove binaries and support files
for dir in /Library/Clawbot /Library/Moltbot \
           /usr/local/bin/clawbot /usr/local/bin/moltbot \
           /var/lib/clawbot /var/lib/moltbot; do
    if [[ -e "$dir" ]]; then
        log "Removing: $dir"
        rm -rf "$dir"
    fi
done

# Remove receipts
for receipt in com.clawbot.* com.moltbot.*; do
    pkgutil --pkgs="$receipt" &>/dev/null && pkgutil --forget "$receipt" 2>/dev/null || true
done

log "Clawbot/moltbot uninstallation completed successfully."
