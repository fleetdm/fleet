#!/bin/bash

# Santa universal extension installer script
# Downloads and installs the latest santa_universal.ext from GitHub

set -e  # Exit on any error

# Variables
GITHUB_REPO="harrisonravazzolo/osquery-santa-extension"
EXTENSION_DIR="/var/fleet/extensions"
OSQUERY_DIR="/var/osquery"
EXTENSIONS_LOAD_FILE="$OSQUERY_DIR/extensions.load"
EXTENSION_NAME="santa_universal.ext"
EXTENSION_PATH="$EXTENSION_DIR/$EXTENSION_NAME"
BACKUP_PATH="$EXTENSION_PATH.backup.$(date +%Y%m%d_%H%M%S)"

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

# Function to get latest release download URL using redirect
get_latest_release_url() {
    # GitHub redirects /releases/latest to the actual latest release page
    local latest_url="https://github.com/$GITHUB_REPO/releases/latest"
    log "Finding latest release URL..."
    
    # Get the redirect location (actual latest release URL)
    local actual_release_url
    if ! actual_release_url=$(curl -s -o /dev/null -w "%{redirect_url}" "$latest_url"); then
        log "Error: Failed to get latest release redirect"
        return 1
    fi
    
    if [[ -z "$actual_release_url" ]]; then
        log "Error: No redirect found for latest release"
        return 1
    fi
    
    log "Latest release URL: $actual_release_url"
    
    # Extract version tag from the URL (e.g., https://github.com/user/repo/releases/tag/v1.0.0)
    local version_tag
    version_tag=$(echo "$actual_release_url" | sed 's|.*/tag/||')
    
    if [[ -z "$version_tag" ]]; then
        log "Error: Could not extract version tag from URL"
        return 1
    fi
    
    log "Found version tag: $version_tag"
    
    # Construct download URL
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version_tag/$EXTENSION_NAME"
    log "Constructed download URL: $download_url"
    
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
    
    local download_url
    if ! download_url=$(get_latest_release_url); then
        log "Error: Failed to get download URL"
        exit 1
    fi
    
    log "Downloading $EXTENSION_NAME..."
    
    # Create temporary file for download
    local temp_file
    temp_file=$(mktemp)
    
    # Download with progress and error handling
    if curl -L --progress-bar --fail -o "$temp_file" "$download_url"; then
        log "Download completed successfully"
        
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
    else
        log "Error: Download failed"
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

# Function to test extension loading
test_extension() {
    log "Testing extension loading..."
    
    # Basic test to see if the extension can be executed
    if "$EXTENSION_PATH" --help &>/dev/null || [[ $? -eq 0 ]]; then
        log "Extension appears to be functional"
    else
        log "Warning: Extension test failed, but this may be normal depending on the extension"
    fi
}

# Function to restart orbit with better error handling
restart_orbit() {
    log "Restarting orbit service..."
    
    # Check if orbit service exists
    if launchctl list | grep -q "com.fleetdm.orbit"; then
        if launchctl kickstart -k system/com.fleetdm.orbit; then
            log "Orbit service restarted successfully"
            
            # Wait a moment and check if service is running
            sleep 2
            if launchctl list | grep -q "com.fleetdm.orbit"; then
                log "Orbit service is running"
            else
                log "Warning: Orbit service may not be running properly"
            fi
        else
            log "Warning: Failed to restart orbit service"
            log "You may need to restart it manually with: sudo launchctl kickstart -k system/com.fleetdm.orbit"
        fi
    else
        log "Warning: Orbit service not found. Extension will load on next orbit startup."
    fi
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
    log "=== Santa Extension Installer Started ==="
    
    check_root
    check_prerequisites
    
    # Create the extensions directory
    create_directory "$EXTENSION_DIR"
    
    # Backup existing extension
    backup_existing
    
    # Download the latest release
    download_latest_release
    
    # Set up file permissions
    setup_file_permissions
    
    # Test the extension
    test_extension
    
    # Setup extensions.load file
    setup_extensions_load
    
    # Restart orbit
    restart_orbit
    
    # Clean up backup on success
    if [[ -f "$BACKUP_PATH" ]]; then
        log "Removing backup file (installation successful)"
        rm -f "$BACKUP_PATH"
    fi
    
    log "=== Installation completed successfully! ==="
    log "Extension installed at: $EXTENSION_PATH"
    log "Extensions configuration: $EXTENSIONS_LOAD_FILE"
    log "The extension should be loaded automatically by osquery/Fleet"
    echo ""
}

# Run the main function
main "$@"