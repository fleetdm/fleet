#!/bin/bash

# Clawbot/moltbot uninstall script for Linux
# Removes Clawbot (also known as moltbot) and cleans up related files

set -e

echo "Starting Clawbot/moltbot uninstallation..."

# Function to log messages with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Ensure running as root
if [ "$(id -u)" -ne 0 ]; then
    log "Error: This script must be run as root"
    exit 1
fi

# Kill any running Clawbot/moltbot processes
log "Stopping Clawbot/moltbot processes..."
pkill -f "clawbot" || true
pkill -f "moltbot" || true
sleep 2

# Stop and disable systemd services
for svc in clawbot moltbot; do
    if systemctl is-active --quiet "$svc" 2>/dev/null; then
        log "Stopping service: $svc"
        systemctl stop "$svc" || true
    fi
    if systemctl is-enabled --quiet "$svc" 2>/dev/null; then
        log "Disabling service: $svc"
        systemctl disable "$svc" || true
    fi
    if [ -f "/etc/systemd/system/${svc}.service" ]; then
        log "Removing service file: /etc/systemd/system/${svc}.service"
        rm -f "/etc/systemd/system/${svc}.service"
    fi
    if [ -f "/usr/lib/systemd/system/${svc}.service" ]; then
        log "Removing service file: /usr/lib/systemd/system/${svc}.service"
        rm -f "/usr/lib/systemd/system/${svc}.service"
    fi
done
systemctl daemon-reload 2>/dev/null || true

# Remove packages via package manager
if command -v dpkg > /dev/null; then
    for pkg in clawbot moltbot; do
        if dpkg -l "$pkg" &>/dev/null; then
            log "Removing deb package: $pkg"
            dpkg --purge "$pkg" || true
        fi
    done
elif command -v rpm > /dev/null; then
    for pkg in clawbot moltbot; do
        if rpm -q "$pkg" &>/dev/null; then
            log "Removing rpm package: $pkg"
            rpm -e "$pkg" || true
        fi
    done
fi

# Remove binaries and directories
for path in /usr/local/bin/clawbot /usr/local/bin/moltbot \
            /opt/clawbot /opt/moltbot \
            /var/lib/clawbot /var/lib/moltbot \
            /var/log/clawbot /var/log/moltbot \
            /etc/clawbot /etc/moltbot; do
    if [ -e "$path" ]; then
        log "Removing: $path"
        rm -rf "$path"
    fi
done

log "Clawbot/moltbot uninstallation completed successfully."
