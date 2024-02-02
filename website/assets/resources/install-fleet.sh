#!/bin/bash

set -e

FLEETCTL_INSTALL_DIR="./.fleetctl/"
FLEETCTL_REPO_URL="https://raw.githubusercontent.com/fleetdm/fleet/main/tools/fleetctl-npm/package.json"
FLEETCTL_BINARY_NAME="fleetctl"


# Check for necessary commands
for cmd in curl tar grep sed; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: $cmd is not installed." >&2
        exit 1
    fi
done

echo "Fetching the latest version of fleetctl..."

# Fetch the latest version number from the GitHub repository

latest_strippedVersion=$(curl -s "https://registry.npmjs.org/fleetctl/latest" | grep -o '"version": *"[^"]*"' | cut -d'"' -f4)
echo "Latest version: $latest_strippedVersion"

# if [[ $latest_strippedVersion != v* ]]; then
#   latest_strippedVersion="v$latest_strippedVersion"
# fi

version_gt() {
  test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1";
}

# Determine OS and Architecture
OS="$(uname -s)"

case "${OS}" in
    Linux*)     OS='linux';;
    Darwin*)    OS='macos';;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

# Function to download and extract fleetctl
download_and_extract() {
    echo "Downloading fleetctl ${latest_strippedVersion} for ${OS}..."
    curl -sSL $DOWNLOAD_URL | tar -xz -C $FLEETCTL_INSTALL_DIR --strip-components=1 ${FLEETCTL_BINARY_NAME}_v${latest_strippedVersion}_${OS}/${FLEETCTL_BINARY_NAME}
}

# Function to check if fleetctl is already installed
check_installed_version() {
    if [ -x "${FLEETCTL_INSTALL_DIR}/${FLEETCTL_BINARY_NAME}" ]; then
        # Extract installed version and remove any 'v' prefix
        installed_version=$("${FLEETCTL_INSTALL_DIR}/${FLEETCTL_BINARY_NAME}" -v | awk 'NR==1{print $NF}' | sed 's/^v//')
        echo "Installed version: ${installed_version}"
    else
        return 1
    fi
}

# Create the install directory if it does not exist.
mkdir -p ${FLEETCTL_INSTALL_DIR}

# Construct download URL
# https://github.com/fleetdm/fleet/releases/download/fleet-v4.43.3/fleetctl_v4.43.3_macos.zip
DOWNLOAD_URL="https://github.com/fleetdm/fleet/releases/download/fleet-v${latest_strippedVersion}/${FLEETCTL_BINARY_NAME}_v${latest_strippedVersion}_${OS}.tar.gz"


if check_installed_version; then
    if version_gt $latest_strippedVersion $installed_version; then
        # Prompt the user for an upgrade
        read -p "A newer version of fleetctl ($latest_strippedVersion) is available. Do you want to upgrade? (y/n): " upgrade_choice

        if [[ "$upgrade_choice" =~ ^[Yy](es)?$ ]]; then
            # Remove the old binary
            rm -f "${FLEETCTL_INSTALL_DIR}/${FLEETCTL_BINARY_NAME}"
            echo "Removed the old version."

            # Download and install the new version
            download_and_extract
            echo "fleetctl installed successfully in ${FLEETCTL_INSTALL_DIR}"
            detect_shell_type_and_update_profile
        else
            echo "Upgrade aborted. Keeping the current version."
        fi
    else
        echo "You already have the latest version of fleetctl (${installed_version}) installed."
    fi
else
    # fleetctl is not present, so download and install it
    download_and_extract
    echo "fleetctl installed successfully in ${FLEETCTL_INSTALL_DIR}"
    detect_shell_type_and_update_profile
fi

# Verify if the binary is executable
if [[ ! -x "${FLEETCTL_INSTALL_DIR}/${FLEETCTL_BINARY_NAME}" ]]; then
    echo "Failed to install or upgrade fleetctl. Please check your permissions or try again later."
    exit 1
fi

# Rest of your script...

# Function to update the user's profile
update_profile() {
    local profile_file="$1"
    local line_to_add="alias fleetctl=\"${FLEETCTL_INSTALL_DIR}/${FLEETCTL_BINARY_NAME}\""

    if ! grep -q "${FLEETCTL_INSTALL_DIR}" "${profile_file}" ; then
        echo "Updating ${profile_file} to include fleetctl in your PATH..."
        echo "${line_to_add}" >> "${profile_file}"
    else
        echo "fleetctl installation path already in ${profile_file}"
    fi
}

detect_shell_type_and_update_profile() {
    # Detect the user's shell and update the corresponding profile file
    case "${SHELL}" in
        */bash)
            update_profile "${HOME}/.bashrc"
            ;;
        */zsh)
            update_profile "${HOME}/.zshrc"
            ;;
        *)
            echo "Unsupported shell: ${SHELL}. Please manually add ${FLEETCTL_INSTALL_DIR} to your PATH."
            ;;
    esac
}
