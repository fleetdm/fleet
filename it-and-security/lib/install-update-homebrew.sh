#!/bin/sh

# Function to check if Homebrew is installed
check_homebrew_installed() {
    if command -v brew >/dev/null 2>&1; then
        echo "Homebrew is already installed."
        return 0
    else
        echo "Homebrew is not installed."
        return 1
    fi
}

# Function to install Homebrew
install_homebrew() {
    echo "Installing Homebrew..."
    /bin/sh -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
}

# Function to update and upgrade Homebrew
update_homebrew() {
    echo "Updating and upgrading Homebrew..."
    brew update
    brew upgrade
}

# Main script
if check_homebrew_installed; then
    update_homebrew
else
    install_homebrew
    if check_homebrew_installed; then
        update_homebrew
    else
        echo "Failed to install Homebrew."
        exit 1
    fi
fi

echo "Homebrew installation and update complete."