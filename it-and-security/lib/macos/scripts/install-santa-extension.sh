#!/bin/bash

# Santa universal extension installer ccript
# Downloads and installs the latest santa_universal.ext from GitHub

set -e  # Exit on any error

# Variables
GITHUB_REPO="harrisonravazzolo/osquery-santa-extension"
EXTENSION_DIR="/var/fleet/extensions"
OSQUERY_DIR="/var/osquery"
EXTENSIONS_LOAD_FILE="$OSQUERY_DIR/extensions.load"
EXTENSION_NAME="santa_universal.ext"
EXTENSION_PATH="$EXTENSION_DIR/$EXTENSION_NAME"

echo "Starting Santa Extension installation..."

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "Error: This script must be run as root (use sudo)"
        exit 1
    fi
}

# Function to create directory with proper ownership
create_directory() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        echo "Creating directory: $dir"
        mkdir -p "$dir"
        chown root:wheel "$dir"
        echo "Directory created and ownership set to root:wheel"
    else
        echo "Directory already exists: $dir"
        # Ensure proper ownership even if directory exists
        chown root:wheel "$dir"
    fi
}

# Function to download the latest release
download_latest_release() {
    echo "Downloading latest $EXTENSION_NAME from GitHub..."
    
    # Use direct download URL for the latest release
    local download_url="https://github.com/$GITHUB_REPO/releases/download/latest/$EXTENSION_NAME"
    
    # Download the file
    echo "Downloading $EXTENSION_NAME..."
    if curl -L -o "$EXTENSION_PATH" "$download_url"; then
        echo "Download completed successfully"
    else
        echo "Error: Download failed. Trying alternative method..."
        
        # Fallback: try to get download URL from GitHub releases page
        echo "Attempting to find download URL from releases page..."
        local releases_page
        releases_page=$(curl -s "https://github.com/$GITHUB_REPO/releases/latest")
        
        if [[ -n "$releases_page" ]]; then
            # Extract the download URL from the HTML
            local alt_download_url
            alt_download_url=$(echo "$releases_page" | grep -o "https://github.com/$GITHUB_REPO/releases/download/[^\"]*/$EXTENSION_NAME" | head -1)
            
            if [[ -n "$alt_download_url" ]]; then
                echo "Found alternative download URL, attempting download..."
                if curl -L -o "$EXTENSION_PATH" "$alt_download_url"; then
                    echo "Download completed successfully using alternative method"
                else
                    echo "Error: Both download methods failed"
                    exit 1
                fi
            else
                echo "Error: Could not find download URL for $EXTENSION_NAME"
                exit 1
            fi
        else
            echo "Error: Could not access GitHub releases page"
            exit 1
        fi
    fi
    
    # Verify the file was downloaded
    if [[ ! -f "$EXTENSION_PATH" ]]; then
        echo "Error: Extension file not found after download"
        exit 1
    fi
    
    # Verify the file is not empty
    if [[ ! -s "$EXTENSION_PATH" ]]; then
        echo "Error: Downloaded file is empty"
        exit 1
    fi
}

# Function to make the extension executable
make_executable() {
    echo "Making extension executable..."
    chmod +x "$EXTENSION_PATH"
    echo "Extension is now executable"
}

# Function to handle extensions.load file
setup_extensions_load() {
    # Create osquery directory if it doesn't exist
    if [[ ! -d "$OSQUERY_DIR" ]]; then
        echo "Creating osquery directory: $OSQUERY_DIR"
        mkdir -p "$OSQUERY_DIR"
        chown root:wheel "$OSQUERY_DIR"
    fi
    
    # Check if extensions.load file exists
    if [[ -f "$EXTENSIONS_LOAD_FILE" ]]; then
        echo "extensions.load file exists, checking for existing entry..."
        
        # Check if the extension path is already in the file
        if grep -q "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE"; then
            echo "Extension path already exists in extensions.load"
        else
            echo "Adding extension path to extensions.load"
            echo "$EXTENSION_PATH" >> "$EXTENSIONS_LOAD_FILE"
        fi
    else
        echo "Creating extensions.load file and adding extension path..."
        echo "$EXTENSION_PATH" > "$EXTENSIONS_LOAD_FILE"
        chown root:wheel "$EXTENSIONS_LOAD_FILE"
    fi
    
    echo "extensions.load file configured"
}

# Function to restart orbit
restart_orbit() {
    echo "Restarting orbit service..."
    if launchctl kickstart -k system/com.fleetdm.orbit; then
        echo "Orbit service restarted successfully"
    else
        echo "Warning: Failed to restart orbit service. You may need to restart it manually."
        echo "Command: sudo launchctl kickstart -k system/com.fleetdm.orbit"
    fi
}

# Main execution
main() {
    check_root
    
    # Create the extensions directory
    create_directory "$EXTENSION_DIR"
    
    # Download the latest release
    download_latest_release
    
    # Make the extension executable
    make_executable
    
    # Setup extensions.load file
    setup_extensions_load
    
    # Restart orbit
    restart_orbit
    
    echo ""
    echo "Installation completed successfully!"
    echo "Extension installed at: $EXTENSION_PATH"
    echo "Extensions configuration: $EXTENSIONS_LOAD_FILE"
    echo ""
}

# Run the main function
main "$@"