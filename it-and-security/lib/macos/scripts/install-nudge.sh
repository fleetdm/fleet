#!/bin/bash
set -e

REPO_OWNER="macadmins"
REPO_NAME="nudge"
DOWNLOAD_DIR="${DOWNLOAD_DIR:-./}"  # Default to current directory, can be overridden
INSTALL_PACKAGE="${INSTALL_PACKAGE:-true}"  # Default to install, can be overridden

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Function to install the package silently
install_package() {
    local filepath="$1"
    
    print_status "Installing Nudge package silently..."
    
    if ! installer -pkg "$filepath" -target /; then
        print_error "Failed to install Nudge package"
        exit 1
    fi
    
    print_success "Nudge package installed successfully"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Downloads and installs the latest Nudge package from GitHub"
    echo ""
    echo "Options:"
    echo "  -d, --dir DIR    Download directory (default: current directory)"
    echo "  -h, --help       Show this help message"
    echo "  --download-only  Download only, do not install"
    echo ""
    echo "Environment variables:"
    echo "  DOWNLOAD_DIR     Override default download directory"
    echo "  INSTALL_PACKAGE  Set to 'false' to download only"
    echo ""
    echo "Examples:"
    echo "  $0                          # Download and install to system"
    echo "  $0 -d /tmp                  # Download to /tmp and install"
    echo "  $0 --download-only          # Download only, do not install"
    echo "  INSTALL_PACKAGE=false $0    # Download only using env var"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            DOWNLOAD_DIR="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        --download-only)
            INSTALL_PACKAGE="false"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

print_status "Starting Nudge download and installation script..."

# Check if running as root (required for installation)
if [[ "$INSTALL_PACKAGE" == "true" ]]; then
    check_root
fi

# Check dependencies
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "jq is required but not installed"
    exit 1
fi

# Get latest release information
print_status "Fetching latest release information..."
api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
release_info=$(curl -s "$api_url")

# Extract version
print_status "Extracting version information..."
tag_name=$(echo "$release_info" | jq -r '.tag_name')

if [ "$tag_name" = "null" ] || [ -z "$tag_name" ]; then
    print_error "Could not extract tag name from release information"
    exit 1
fi

# Remove 'v' prefix if present
version=$(echo "$tag_name" | sed 's/^v//')
print_status "Latest version: v${version}"

# Construct download URL
download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/v${version}/Nudge-${version}.pkg"
filename="Nudge-${version}.pkg"
filepath="${DOWNLOAD_DIR}/${filename}"

print_status "Downloading Nudge v${version}..."
print_status "URL: $download_url"
print_status "Destination: $filepath"

# Create download directory if it doesn't exist
mkdir -p "$DOWNLOAD_DIR"

# Download with progress bar and follow redirects
if curl -L --progress-bar -o "$filepath" "$download_url"; then
    print_success "Downloaded: $filepath"
    
    # Display file information
    if [ -f "$filepath" ]; then
        file_size=$(ls -lh "$filepath" | awk '{print $5}')
        print_status "File size: $file_size"
    fi
else
    print_error "Failed to download $filename"
    exit 1
fi

# Install the package if requested
if [[ "$INSTALL_PACKAGE" == "true" ]]; then
    install_package "$filepath"
    print_success "Nudge v${version} downloaded and installed successfully!"
else
    print_success "Nudge v${version} downloaded successfully!"
    print_status "Package location: $filepath"
    print_status "Run 'sudo installer -pkg \"$filepath\" -target /' to install manually"
fi
