#!/bin/bash

# Santa universal extension installer script
# Downloads and installs the latest santa_universal.ext from GitHub
# Safe for deployment via Fleet (schedules orbit restart to avoid script termination)
#
# Usage:
#   sudo ./install_santa_extension.sh          # Default: scheduled restart (Fleet-safe)
#   sudo ./install_santa_extension.sh immediate # Immediate restart (manual execution)

set -e  # Exit on any error

# Variables
GITHUB_REPO="allenhouchins/fleet-extensions"
EXTENSION_DIR="/var/fleet/extensions"
OSQUERY_DIR="/var/osquery"
EXTENSIONS_LOAD_FILE="$OSQUERY_DIR/extensions.load"
EXTENSION_NAME="santa.ext"
EXTENSION_PATH="$EXTENSION_DIR/$EXTENSION_NAME"
BACKUP_PATH="$EXTENSION_PATH.backup.$(date +%Y%m%d_%H%M%S)"

# Command line options
IMMEDIATE_RESTART=${1:-false}  # Pass "immediate" as first argument for immediate restart

echo "Starting Santa Extension installation..."

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

# Function to check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        log "Error: curl is required but not installed"
        exit 1
    fi
    
    log "Prerequisites check completed"
}

# Function to create directory with proper ownership
create_directory() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        log "Creating directory: $dir"
        mkdir -p "$dir"
        chown root:wheel "$dir"
        chmod 755 "$dir"
        log "Directory created with proper permissions"
    else
        log "Directory already exists: $dir"
        # Ensure proper ownership even if directory exists
        chown root:wheel "$dir"
        chmod 755 "$dir"
    fi
}

# Function to backup existing extension
backup_existing() {
    if [[ -f "$EXTENSION_PATH" ]]; then
        log "Backing up existing extension to: $BACKUP_PATH"
        cp "$EXTENSION_PATH" "$BACKUP_PATH"
        log "Backup completed"
    fi
}

# Function to get the latest release tag from GitHub
get_latest_release_tag() {
    log "Finding latest release tag..."
    
    # Try to get the latest release page and extract the actual tag
    local releases_url="https://github.com/$GITHUB_REPO/releases/latest"
    local response
    
    if ! response=$(curl -s -L "$releases_url"); then
        log "Error: Failed to fetch releases page"
        return 1
    fi
    
    # Extract the actual tag from the redirected URL or page content
    # Look for the tag in the URL path or in the page content
    local tag
    tag=$(echo "$response" | grep -o 'releases/tag/[^"]*' | head -1 | sed 's|releases/tag/||' | sed 's|".*||')
    
    if [[ -z "$tag" ]]; then
        # Alternative: look for version tags in the page content
        tag=$(echo "$response" | grep -o 'tag/[v0-9][^"]*' | head -1 | sed 's|tag/||' | sed 's|".*||')
    fi
    
    if [[ -z "$tag" ]]; then
        log "Error: Could not determine latest release tag"
        return 1
    fi
    
    log "Found latest release tag: $tag"
    echo "$tag"
}

# Function to construct download URL with specific tag
get_download_url_with_tag() {
    local tag="$1"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$tag/$EXTENSION_NAME"
    echo "$download_url"
}

# Function to validate downloaded file
validate_download() {
    local file_path="$1"
    
    log "Validating downloaded file..."
    
    # Check if file exists and is not empty
    if [[ ! -f "$file_path" ]]; then
        log "Error: Downloaded file not found"
        return 1
    fi
    
    if [[ ! -s "$file_path" ]]; then
        log "Error: Downloaded file is empty"
        return 1
    fi
    
    # Check if file is executable format (basic check)
    local file_type
    file_type=$(file "$file_path" 2>/dev/null || echo "unknown")
    log "File type: $file_type"
    
    # For macOS, check if it's a Mach-O executable
    if [[ "$file_type" == *"Mach-O"* ]] || [[ "$file_type" == *"executable"* ]]; then
        log "File validation passed"
        return 0
    else
        log "Warning: File may not be a valid executable. Proceeding anyway..."
        return 0
    fi
}

# Function to download the latest release
download_latest_release() {
    log "Starting download process..."
    
    # Create temporary file for download
    local temp_file
    temp_file=$(mktemp)
    
    # First, try the direct latest download URL
    local direct_url="https://github.com/$GITHUB_REPO/releases/latest/download/$EXTENSION_NAME"
    log "Attempting direct download from: $direct_url"
    
    if curl -L --progress-bar --fail -o "$temp_file" "$direct_url" 2>/dev/null; then
        log "Direct download successful"
    else
        log "Direct download failed, getting actual release tag..."
        
        # Get the actual latest release tag
        local latest_tag
        if ! latest_tag=$(get_latest_release_tag); then
            log "Error: Could not determine latest release tag"
            rm -f "$temp_file"
            exit 1
        fi
        
        # Construct download URL with the actual tag
        local download_url
        download_url=$(get_download_url_with_tag "$latest_tag")
        log "Download URL with tag: $download_url"
        
        # Download with the specific tag
        if curl -L --progress-bar --fail -o "$temp_file" "$download_url"; then
            log "Download with specific tag successful"
        else
            log "Error: Download failed with both methods"
            log "Please verify that '$EXTENSION_NAME' exists in the latest release at:"
            log "https://github.com/$GITHUB_REPO/releases/latest"
            rm -f "$temp_file"
            exit 1
        fi
    fi
    
    # Validate the download
    if validate_download "$temp_file"; then
        # Move to final location
        mv "$temp_file" "$EXTENSION_PATH"
        log "File moved to final location: $EXTENSION_PATH"
    else
        log "Error: File validation failed"
        rm -f "$temp_file"
        exit 1
    fi
}

# Function to make the extension executable and set proper ownership
setup_file_permissions() {
    log "Setting up file permissions..."
    chown root:wheel "$EXTENSION_PATH"
    chmod 755 "$EXTENSION_PATH"
    log "File permissions configured (owner: root:wheel, mode: 755)"
}

# Function to handle extensions.load file
setup_extensions_load() {
    log "Configuring extensions.load file..."
    
    # Create osquery directory if it doesn't exist
    if [[ ! -d "$OSQUERY_DIR" ]]; then
        log "Creating osquery directory: $OSQUERY_DIR"
        mkdir -p "$OSQUERY_DIR"
        chown root:wheel "$OSQUERY_DIR"
        chmod 755 "$OSQUERY_DIR"
    fi
    
    # Check if extensions.load file exists
    if [[ -f "$EXTENSIONS_LOAD_FILE" ]]; then
        log "extensions.load file exists, checking for existing entry..."
        
        # Remove any existing entries for this extension (handle duplicates)
        if grep -q "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE"; then
            log "Removing existing entries for this extension..."
            # Create temp file without the extension path
            grep -v "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE" > "$EXTENSIONS_LOAD_FILE.tmp" || true
            mv "$EXTENSIONS_LOAD_FILE.tmp" "$EXTENSIONS_LOAD_FILE"
        fi
        
        # Add the extension path
        echo "$EXTENSION_PATH" >> "$EXTENSIONS_LOAD_FILE"
        log "Extension path added to extensions.load"
    else
        log "Creating extensions.load file..."
        echo "$EXTENSION_PATH" > "$EXTENSIONS_LOAD_FILE"
        chown root:wheel "$EXTENSIONS_LOAD_FILE"
        chmod 644 "$EXTENSIONS_LOAD_FILE"
        log "extensions.load file created"
    fi
}

# Function to remove old extension and its reference from extensions.load
remove_old_extension() {
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
    log "Check /var/log/santa_installer.log for restart status"
}

# Function to handle the detached restart process
handle_detached_restart() {
    # This runs in the detached child process
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting detached restart process..." >> /var/log/santa_installer.log 2>&1
    
    # Wait for parent process to complete and report success
    sleep 10
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Executing orbit restart..." >> /var/log/santa_installer.log 2>&1
    launchctl kickstart -k system/com.fleetdm.orbit >> /var/log/santa_installer.log 2>&1
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Orbit restart command executed" >> /var/log/santa_installer.log 2>&1
}

# Function to cleanup on failure
cleanup_on_failure() {
    log "Cleaning up due to failure..."
    
    # Remove the downloaded extension if it exists
    if [[ -f "$EXTENSION_PATH" ]]; then
        rm -f "$EXTENSION_PATH"
        log "Removed failed installation file"
    fi
    
    # Restore backup if it exists
    if [[ -f "$BACKUP_PATH" ]]; then
        mv "$BACKUP_PATH" "$EXTENSION_PATH"
        log "Restored previous version from backup"
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
    
    log "=== Santa Extension Installer Started ==="
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Mode: Immediate restart (manual execution)"
    else
        log "Mode: Scheduled restart (safe for Fleet deployment)"
    fi
    
    # Ensure log directory exists for background process
    mkdir -p /var/log
    
    check_root
    check_prerequisites
    
    # Create the extensions directory
    create_directory "$EXTENSION_DIR"
    
    # Remove old extension and its reference before proceeding
    remove_old_extension
    
    # Backup existing extension
    backup_existing
    
    # Download the latest release
    download_latest_release
    
    # Set up file permissions
    setup_file_permissions

    # Setup extensions.load file
    setup_extensions_load
    
    # Handle orbit restart (scheduled for Fleet deployment, immediate for manual)
    handle_orbit_restart
    
    # Clean up backup on success
    if [[ -f "$BACKUP_PATH" ]]; then
        log "Removing backup file (installation successful)"
        rm -f "$BACKUP_PATH"
    fi
    
    log "=== Installation completed successfully! ==="
    log "Extension installed at: $EXTENSION_PATH"
    log "Extensions configuration: $EXTENSIONS_LOAD_FILE"
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Orbit service has been restarted immediately"
    else
        log "Orbit service restart has been scheduled for 10 seconds"
    fi
    echo ""
}

# Run the main function
main "$@"