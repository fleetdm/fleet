#!/bin/bash

set -e

FLEETCTL_INSTALL_DIR="${HOME}/.fleetctl/"
FLEETCTL_BINARY_NAME="fleetctl"


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
    Linux*)     OS='linux';;
    Darwin*)    OS='macos';;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

# Download the fleetctl binary and extract it into the install directory
download_and_extract() {
    echo "Downloading fleetctl ${latest_strippedVersion} for ${OS}..."
    curl -sSL $DOWNLOAD_URL | tar -xz -C $FLEETCTL_INSTALL_DIR --strip-components=1 fleetctl_v${latest_strippedVersion}_${OS}/fleetctl
}

# Check to see if the fleetctl binary exists in the script's install directory.
check_installed_version() {
    # If the fleetctl binary exists, we'll check the version of it using fleetctl -v.
    if [ -x "${FLEETCTL_INSTALL_DIR}/fleetctl" ]; then
        installed_version=$("${FLEETCTL_INSTALL_DIR}/fleetctl" -v | awk 'NR==1{print $NF}' | sed 's/^v//')
        echo "Installed version: ${installed_version}"
    else
        return 1
    fi
}

# Create the install directory if it does not exist.
mkdir -p ${FLEETCTL_INSTALL_DIR}

# Construct download URL
# ex: https://github.com/fleetdm/fleet/releases/download/fleet-v4.43.3/fleetctl_v4.43.3_macos.zip
DOWNLOAD_URL="https://github.com/fleetdm/fleet/releases/download/fleet-v${latest_strippedVersion}/fleetctl_v${latest_strippedVersion}_${OS}.tar.gz"


if check_installed_version; then
    if version_gt $latest_strippedVersion $installed_version; then
        # Prompt the user for an upgrade
        read -p "A newer version of fleetctl ($latest_strippedVersion) is available. Do you want to upgrade? (y/n): " upgrade_choice

        if [[ "$upgrade_choice" =~ ^[Yy](es)?$ ]]; then
            # Remove the old binary
            rm -f "${FLEETCTL_INSTALL_DIR}/fleetctl"
            echo "Removed the old version."

            # Download and install the new version
            download_and_extract
            echo "fleetctl installed successfully in ${FLEETCTL_INSTALL_DIR}"
        else
            echo "Upgrade aborted. Keeping the current version."
        fi
    else
        echo "You already have the latest version of fleetctl (${installed_version}) installed."
    fi
else
    # If there is no existing fleetctl binary, download the latest version and extract it.
    download_and_extract
    echo "fleetctl installed successfully in ${FLEETCTL_INSTALL_DIR}"
fi

# Verify if the binary is executable
if [[ ! -x "${FLEETCTL_INSTALL_DIR}/fleetctl" ]]; then
    echo "Failed to install or upgrade fleetctl. Please check your permissions or try again later."
    exit 1
fi

