#!/usr/bin/env bash

# set -x
# trap read debug

set -eo pipefail

# Run this script in user context (not root)
if [ "$EUID" = 0 ]
then
	printf "To prevent unnecessary privilege elevation do not execute this script as the root user.\nExiting..."; exit 1
fi

# Reference: https://wiki.winehq.org/MacOS
# Wine can be installed without brew via a distribution such as https://github.com/Gcenx/macOS_Wine_builds/releases/tag/9.0, or by building from source.
# Check if brew is installed
if ! command -v brew >/dev/null 2>&1
then
    printf "Homebrew is not installed.\nPlease install Homebrew.\nFor instructions, see https://brew.sh/"; exit 1
fi

# Install wine via brew with warning
printf "\nWARNING: The Wine app developer has an Apple Developer certificate but the\napp bundle post-installation will not be code-signed or notarized.\n\nDo you wish to proceed?\n\n"
while true
do
    read -r -p "install> " install
    case "$install" in
        y|yes|Y|YES ) brew install --cask --no-quarantine https://raw.githubusercontent.com/Homebrew/homebrew-cask/1ecfe82f84e0f3c3c6b741d3ddc19a164c2cb18d/Casks/w/wine-stable.rb ;;
          n|no|N|NO ) printf "\nExiting...\n\n"; exit 1 ;;
                  * ) printf "\nPlease enter yes or no at the prompt...\n\n" ;;
    esac
done
