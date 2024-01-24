#!/usr/bin/env bash

set -eo pipefail

# Check if brew is installed
if ! command -v brew >/dev/null 2>&1 ; then
    echo "Homebrew is not installed. Please install Homebrew first. For instructions, see https://brew.sh/"
    exit 1
fi

# Installing wine 9.0
brew install --cask --no-quarantine https://raw.githubusercontent.com/Homebrew/homebrew-cask/1ecfe82f84e0f3c3c6b741d3ddc19a164c2cb18d/Casks/w/wine-stable.rb
