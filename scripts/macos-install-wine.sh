#!/usr/bin/env bash


set -eo pipefail


brew_wine(){
# Wine reference: https://wiki.winehq.org/MacOS
# Wine can be installed without brew via a distribution such as https://github.com/Gcenx/macOS_Wine_builds/releases/tag/9.0 or by building from source.
curl -O https://raw.githubusercontent.com/Homebrew/homebrew-cask/1ecfe82f84e0f3c3c6b741d3ddc19a164c2cb18d/Casks/w/wine-stable.rb
brew install --cask --no-quarantine wine-stable.rb; exit 0
}


warn_wine(){
printf "\nWARNING: The Wine app developer has an Apple Developer certificate but the\napp bundle post-installation will not be code-signed or notarized.\n\nDo you wish to proceed?\n\n"
while true
do
    read -r -p "install> " install
    case "$install" in
        y|yes|Y|YES) brew_wine ;;
          n|no|N|NO) printf "\nExiting...\n\n"; exit 1 ;;
                  *) printf "\nPlease enter yes or no at the prompt...\n\n" ;;
    esac
done
}


# option to execute script in non-interactive mode
while getopts 'n' option
do
    case "$option" in
        n) mode=auto ;;
        *) : ;;
    esac
done


# prevent root execution
if [ "$EUID" = 0 ]
then
    printf "\nTo prevent unnecessary privilege elevation do not execute this script as the root user.\nExiting...\n\n"; exit 1
fi


# check if Homebrew is installed
if ! command -v brew > /dev/null 2>&1
then
    printf "\nHomebrew is not installed.\nPlease install Homebrew.\nFor instructions, see https://brew.sh/\n\n"; exit 1
fi


# install Wine
if [ "$mode" = 'auto' ]
then
    printf "\n%s executed in non-interactive mode.\n\n" "$0"; brew_wine
else
    warn_wine
fi

