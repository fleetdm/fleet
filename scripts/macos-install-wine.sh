#!/usr/bin/env bash

set -eo pipefail

# Run this script in user context (not root).
# Reference: https://wiki.winehq.org/MacOS
# Wine can be installed without brew via a distribution such as https://github.com/Gcenx/macOS_Wine_builds/releases/tag/9.0, or by building from source.

# Check if brew is installed
if ! command -v brew >/dev/null 2>&1 ; then
    echo "Homebrew is not installed. Please install Homebrew first. For instructions, see https://brew.sh/"
    exit 1
fi

# Install wine via brew
brew install --cask --no-quarantine https://raw.githubusercontent.com/Homebrew/homebrew-cask/1ecfe82f84e0f3c3c6b741d3ddc19a164c2cb18d/Casks/w/wine-stable.rb
