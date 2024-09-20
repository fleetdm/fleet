#!/bin/bash

set -e

FLEETCTL_INSTALL_DIR="${HOME}/.fleetctl/"


# Check for necessary commands
for cmd in curl tar grep sed; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: $cmd is not installed." >&2
        exit 1
    fi
done

echo "Fetching the latest version of fleetctl..."


# Fetch the latest version number from NPM
latest_strippedVersion=$(curl -s "https://registry.npmjs.org/fleetctl/latest" | grep -o '"version": *"[^"]*"' | cut -d'"' -f4)
echo "Latest version available on NPM: $latest_strippedVersion"

version_gt() {
  test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1";
}

# Determine operating system (Linux or MacOS)
OS="$(uname -s)"

case "${OS}" in
    Linux*)     OS='linux' OS_DISPLAY_NAME='Linux';;
    Darwin*)    OS='macos' OS_DISPLAY_NAME='macOS';;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

# Create the install directory if it does not exist.
mkdir -p "${FLEETCTL_INSTALL_DIR}"

# Construct download URL
# ex: https://github.com/fleetdm/fleet/releases/download/fleet-v4.43.3/fleetctl_v4.43.3_macos.zip
DOWNLOAD_URL="https://github.com/fleetdm/fleet/releases/download/fleet-v${latest_strippedVersion}/fleetctl_v${latest_strippedVersion}_${OS}.tar.gz"

# Download the latest version of fleetctl and extract it.
echo "Downloading fleetctl ${latest_strippedVersion} for ${OS_DISPLAY_NAME}..."
curl -sSL "$DOWNLOAD_URL" | tar -xz -C "$FLEETCTL_INSTALL_DIR" --strip-components=1 fleetctl_v"${latest_strippedVersion}"_${OS}/
echo "fleetctl installed successfully in ${FLEETCTL_INSTALL_DIR}"
echo
echo "To start the local demo:"
echo
echo "1. Start Docker Desktop"
echo "2. To access your Fleet Premium trial, head to fleetdm.com/try-fleet and run the command in step 2."

# Verify if the binary is executable
if [[ ! -x "${FLEETCTL_INSTALL_DIR}/fleetctl" ]]; then
    echo "Failed to install or upgrade fleetctl. Please check your permissions and try running this script again."
    exit 1
fi
