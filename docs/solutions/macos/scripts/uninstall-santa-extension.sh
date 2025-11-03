#!/bin/bash

# Santa universal extension uninstaller script
# Removes the santa.ext extension and cleans up configuration
# Safe for deployment via Fleet (schedules orbit restart to avoid script termination)
#
# Usage:
#   sudo ./uninstall-santa-extension.sh          # Default: scheduled restart (Fleet-safe)
#   sudo ./uninstall-santa-extension.sh immediate # Immediate restart (manual execution)

set -e  # Exit on any error

# Variables
EXTENSION_DIR="/var/fleet/extensions"
OSQUERY_DIR="/var/osquery"
EXTENSIONS_LOAD_FILE="$OSQUERY_DIR/extensions.load"
EXTENSION_NAME="santa.ext"
EXTENSION_PATH="$EXTENSION_DIR/$EXTENSION_NAME"
BACKUP_PATH="$EXTENSION_PATH.backup.$(date +%Y%m%d_%H%M%S)"

# Command line options
IMMEDIATE_RESTART=${1:-false}  # Pass "immediate" as first argument for immediate restart

echo "Starting Santa Extension uninstallation..."

# Function to log messages with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log "Error: This script must be run as root (use sudo)"
        exit 1
    fi
}

# Function to check if extension exists
check_extension_exists() {
    if [[ ! -f "$EXTENSION_PATH" ]]; then
        log "Warning: Extension file not found at $EXTENSION_PATH"
        log "Extension may already be uninstalled or was never installed"
        return 1
    fi
    return 0
}

# Function to backup extension before removal (optional safety measure)
backup_extension() {
    if [[ -f "$EXTENSION_PATH" ]]; then
        log "Creating backup of extension before removal: $BACKUP_PATH"
        cp "$EXTENSION_PATH" "$BACKUP_PATH"
        log "Backup completed"
    fi
}

# Function to remove extension file
remove_extension_file() {
    if [[ -f "$EXTENSION_PATH" ]]; then
        log "Removing extension file: $EXTENSION_PATH"
        rm -f "$EXTENSION_PATH"
        log "Extension file removed successfully"
    else
        log "Extension file not found, skipping removal"
    fi
}

# Function to remove extension reference from extensions.load file
remove_extension_reference() {
    log "Updating extensions.load file..."
    
    # Check if extensions.load file exists
    if [[ ! -f "$EXTENSIONS_LOAD_FILE" ]]; then
        log "extensions.load file not found, skipping reference removal"
        return 0
    fi
    
    # Check if the extension path exists in the file
    if grep -q "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE"; then
        log "Removing extension reference from extensions.load..."
        
        # Create backup of extensions.load file
        cp "$EXTENSIONS_LOAD_FILE" "$EXTENSIONS_LOAD_FILE.backup.$(date +%Y%m%d_%H%M%S)"
        
        # Remove the extension path line
        grep -v "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE" > "$EXTENSIONS_LOAD_FILE.tmp" || true
        mv "$EXTENSIONS_LOAD_FILE.tmp" "$EXTENSIONS_LOAD_FILE"
        
        log "Extension reference removed from extensions.load"
        
        # Check if the file is now empty and remove it if so
        if [[ ! -s "$EXTENSIONS_LOAD_FILE" ]]; then
            log "extensions.load file is now empty, removing it"
            rm -f "$EXTENSIONS_LOAD_FILE"
        fi
    else
        log "Extension reference not found in extensions.load file"
    fi
}

# Function to remove old extension references (for backward compatibility)
remove_old_extension_references() {
    local old_extension_name="santa_universal.ext"
    local old_extension_path="$EXTENSION_DIR/$old_extension_name"
    
    # Remove old extension file if it exists
    if [[ -f "$old_extension_path" ]]; then
        log "Removing old extension file: $old_extension_path"
        rm -f "$old_extension_path"
    fi
    
    # Remove old extension reference from extensions.load if it exists
    if [[ -f "$EXTENSIONS_LOAD_FILE" ]]; then
        if grep -q "$old_extension_path" "$EXTENSIONS_LOAD_FILE"; then
            log "Removing old extension reference from extensions.load"
            grep -v "$old_extension_path" "$EXTENSIONS_LOAD_FILE" > "$EXTENSIONS_LOAD_FILE.tmp" || true
            mv "$EXTENSIONS_LOAD_FILE.tmp" "$EXTENSIONS_LOAD_FILE"
        fi
    fi
}

# Function to schedule orbit restart using detached child process or restart immediately
handle_orbit_restart() {
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Immediate restart requested - restarting orbit service now..."
        restart_orbit_immediate
    else
        log "Scheduling orbit service restart (safe for Fleet deployment)..."
        schedule_orbit_restart
    fi
}

# Function to restart orbit immediately 
restart_orbit_immediate() {
    log "Restarting orbit service immediately..."
    launchctl kickstart -k system/com.fleetdm.orbit
    log "Orbit service restart command executed"
}

# Function to schedule orbit restart using detached child process
schedule_orbit_restart() {
    log "Scheduling orbit restart in 10 seconds (detached process method)..."
    
    # Start detached child process that will handle the restart
    bash -c "bash $0 __restart_orbit >/dev/null 2>/dev/null </dev/null &"
    
    log "Orbit restart scheduled for 10 seconds after script completion"
    log "Check /var/log/santa_uninstaller.log for restart status"
}

# Function to handle the detached restart process
handle_detached_restart() {
    # This runs in the detached child process
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting detached restart process..." >> /var/log/santa_uninstaller.log 2>&1
    
    # Wait for parent process to complete and report success
    sleep 10
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Executing orbit restart..." >> /var/log/santa_uninstaller.log 2>&1
    launchctl kickstart -k system/com.fleetdm.orbit >> /var/log/santa_uninstaller.log 2>&1
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Orbit restart command executed" >> /var/log/santa_uninstaller.log 2>&1
}

# Function to cleanup on failure
cleanup_on_failure() {
    log "Cleaning up due to failure..."
    
    # Restore backup if it exists
    if [[ -f "$BACKUP_PATH" ]]; then
        mv "$BACKUP_PATH" "$EXTENSION_PATH"
        log "Restored extension from backup"
    fi
}

# Trap to handle errors
trap cleanup_on_failure ERR

# Main execution
main() {
    # Handle detached restart process
    if [[ "$1" == "__restart_orbit" ]]; then
        handle_detached_restart
        exit 0
    fi
    
    log "=== Santa Extension Uninstaller Started ==="
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Mode: Immediate restart (manual execution)"
    else
        log "Mode: Scheduled restart (safe for Fleet deployment)"
    fi
    
    # Ensure log directory exists for background process
    mkdir -p /var/log
    
    check_root
    
    # Check if extension exists
    if ! check_extension_exists; then
        log "Extension not found. Checking for any remaining references..."
        remove_extension_reference
        remove_old_extension_references
        
        # Still restart orbit to ensure clean state
        handle_orbit_restart
        
        log "=== Uninstallation completed (extension was not installed) ==="
        exit 0
    fi
    
    # Create backup of extension before removal
    backup_extension
    
    # Remove extension file
    remove_extension_file
    
    # Remove extension reference from extensions.load
    remove_extension_reference
    
    # Remove any old extension references
    remove_old_extension_references
    
    # Handle orbit restart (scheduled for Fleet deployment, immediate for manual)
    handle_orbit_restart
    
    # Clean up backup on success
    if [[ -f "$BACKUP_PATH" ]]; then
        log "Removing backup file (uninstallation successful)"
        rm -f "$BACKUP_PATH"
    fi
    
    log "=== Uninstallation completed successfully! ==="
    log "Extension removed from: $EXTENSION_PATH"
    log "Extension reference removed from: $EXTENSIONS_LOAD_FILE"
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Orbit service has been restarted immediately"
    else
        log "Orbit service restart has been scheduled for 10 seconds"
    fi
    echo ""
}

# Run the main function
main "$@"
