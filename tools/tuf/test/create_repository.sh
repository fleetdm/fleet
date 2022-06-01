#!/bin/bash

set -e

# This script initializes a test Fleet TUF repository.
# All targets are created with version 42.

# Input:
# TUF_PATH: directory path for the test TUF repository.
# FLEET_ROOT_PASSPHRASE: Root role passphrase.
# FLEET_TARGETS_PASSPHRASE: Targets role passphrase.
# FLEET_SNAPSHOT_PASSPHRASE: Snapshot role passphrase.
# FLEET_TIMESTAMP_PASSPHRASE: Timestamp role passphrase.
# SYSTEMS: Space separated list of systems to support in the TUF repository. Default value is: "macos windows linux"

if [[ -z "$TUF_PATH" ]]; then
    echo "Must set the TUF_PATH environment variable."
    exit 1
fi
if [[ -d "$TUF_PATH" ]]; then
    echo "$TUF_PATH directory already exists, nothing to do."
    exit 0
fi

OSQUERY_MACOS_APP_BUNDLE_VERSION=5.2.3
SYSTEMS=${SYSTEMS:-macos linux windows}

mkdir -p $TUF_PATH/tmp

./build/fleetctl updates init --path $TUF_PATH

for system in $SYSTEMS; do

    if [[ $system == "macos" ]]; then
        # Generate and add osqueryd .app bundle for macos-app.
        osqueryd_path=$TUF_PATH/tmp/osqueryd.app.tar.gz
        make osqueryd-app-tar-gz version=$OSQUERY_MACOS_APP_BUNDLE_VERSION out-path=$(dirname $osqueryd_path)
        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target $osqueryd_path \
            --platform macos-app \
            --name osqueryd \
            --version 42.0.0 -t 42.0 -t 42 -t stable
        rm $osqueryd_path
    else
        # Use latest stable version of osqueryd from our TUF server.
        osqueryd="osqueryd"
        if [[ $system == "windows" ]]; then
            osqueryd="$osqueryd.exe"
        fi
        osqueryd_path="$TUF_PATH/tmp/$osqueryd"
        curl https://tuf.fleetctl.com/targets/osqueryd/$system/stable/$osqueryd --output $osqueryd_path

        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target $osqueryd_path \
            --platform $system \
            --name osqueryd \
            --version 42.0.0 -t 42.0 -t 42 -t stable
        rm $osqueryd_path
    fi

    goose_value="$system"
    if [[ $system == "macos" ]]; then
        goose_value="darwin"
    fi
    orbit_target=orbit-$system
    if [[ $system == "windows" ]]; then
        orbit_target="${orbit_target}.exe"
    fi

    # Compile the latest version of orbit from source.
    GOOS=$goose_value GOARCH=amd64 go build -o $orbit_target ./orbit/cmd/orbit

    # If macOS and CODESIGN_IDENTITY is defined, sign the executable.
    if [[ $system == "macos" && -n "$CODESIGN_IDENTITY" ]]; then
        codesign -s "$CODESIGN_IDENTITY" -i com.fleetdm.orbit -f -v --timestamp --options runtime $orbit_target
    fi

    ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target $orbit_target \
        --platform $system \
        --name orbit \
        --version 42.0.0 -t 42.0 -t 42 -t stable
    rm $orbit_target

    # Add Fleet Desktop application on macos (if enabled).
    if [[ $system == "macos" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERBOSE=1 \
        FLEET_DESKTOP_VERSION=42.0.0 \
        make desktop-app-tar-gz
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target desktop.app.tar.gz \
        --platform macos \
        --name desktop \
        --version 42.0.0 -t 42.0 -t 42 -t stable
        rm desktop.app.tar.gz
    fi

    # Add Fleet Desktop application on windows (if enabled).
    if [[ $system == "windows" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=42.0.0 \
        make desktop-windows
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target fleet-desktop.exe \
        --platform windows \
        --name desktop \
        --version 42.0.0 -t 42.0 -t 42 -t stable
        rm fleet-desktop.exe
    fi

    # Add Fleet Desktop application on  (if enabled).
    if [[ $system == "linux" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=42.0.0 \
        make desktop-linux
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target desktop.tar.gz \
        --platform linux \
        --name desktop \
        --version 42.0.0 -t 42.0 -t 42 -t stable
        rm desktop.tar.gz
    fi
done
