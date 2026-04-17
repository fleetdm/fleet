#!/bin/bash

# Please don't delete. This script is linked to, as a redirect, from fleetctl and the Fleet website.

set -eo pipefail


# Wine reference: https://wiki.winehq.org/MacOS
# We install from the Gcenx Wine builds tarball directly (rather than via
# homebrew-cask) because the wine-stable / wine-devel casks either don't exist
# or are deprecated, and because pinning a homebrew-cask commit breaks whenever
# Gcenx removes an older release. See https://github.com/fleetdm/fleet/issues/43484.
WINE_VERSION="11.6_1"
WINE_VARIANT="wine-devel"
WINE_TARBALL="${WINE_VARIANT}-${WINE_VERSION}-osx64.tar.xz"
WINE_URL="https://github.com/Gcenx/macOS_Wine_builds/releases/download/${WINE_VERSION}/${WINE_TARBALL}"
WINE_SHA256="737c5bbcef4dab626e6dcb58f3736bf910780de25d53e52e7d7c01e521da725a"
WINE_APP="/Applications/Wine Devel.app"


install_wine(){
    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' EXIT

    printf "Downloading %s...\n" "${WINE_TARBALL}"
    curl -fsSL --retry 3 -o "${tmpdir}/${WINE_TARBALL}" "${WINE_URL}"

    printf "Verifying sha256...\n"
    echo "${WINE_SHA256}  ${tmpdir}/${WINE_TARBALL}" | shasum -a 256 -c -

    printf "Extracting to /Applications...\n"
    if [ -e "${WINE_APP}" ]; then
        rm -rf "${WINE_APP}"
    fi
    tar -xJf "${tmpdir}/${WINE_TARBALL}" -C /Applications

    printf "Stripping quarantine attribute...\n"
    xattr -dr com.apple.quarantine "${WINE_APP}" || true

    # Put the wine binary on PATH via the Homebrew prefix (already on PATH for
    # brew users) so callers like `fleetctl package --type msi` can find it.
    brew_bin="$(brew --prefix)/bin"
    mkdir -p "${brew_bin}"
    ln -sf "${WINE_APP}/Contents/Resources/wine/bin/wine" "${brew_bin}/wine"

    printf "Wine installed: "
    "${brew_bin}/wine" --version
    exit 0
}


warn_wine(){
printf "\nWARNING: The Wine app developer has an Apple Developer certificate but the\napp bundle post-installation will not be code-signed or notarized.\n\nDo you wish to proceed?\n\n"
while true
do
    read -r -p "install> " install
    case "$install" in
        y|yes|Y|YES) install_wine ;;
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


# check if Homebrew is installed (used to locate a directory on PATH for the wine symlink)
if ! command -v brew > /dev/null 2>&1
then
    printf "\nHomebrew is not installed.\nPlease install Homebrew.\nFor instructions, see https://brew.sh/\n\n"; exit 1
fi


# install Wine
if [ "$mode" = 'auto' ]
then
    printf "\n%s executed in non-interactive mode.\n\n" "$0"; install_wine
else
    warn_wine
fi
