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
# MACOS_USE_PREBUILT_DESKTOP_APP_TAR_GZ: Set variable to use a pre-built desktop.app.tar.gz. Useful when running on non-macOS host.

if [[ -z "$TUF_PATH" ]]; then
    echo "Must set the TUF_PATH environment variable."
    exit 1
fi
if [[ -d "$TUF_PATH" ]]; then
    echo "$TUF_PATH directory already exists, nothing to do."
    exit 0
fi

SYSTEMS=${SYSTEMS:-macos linux windows}
NUDGE_VERSION=stable
SWIFT_DIALOG_MACOS_APP_VERSION=2.2.1
SWIFT_DIALOG_MACOS_APP_BUILD_VERSION=4591

if [[ -z "$OSQUERY_VERSION" ]]; then
    OSQUERY_VERSION=5.11.0
fi

mkdir -p $TUF_PATH/tmp

./build/fleetctl updates init --path $TUF_PATH

for system in $SYSTEMS; do

    # Use latest stable version of osqueryd from our TUF server.
    osqueryd="osqueryd"
    osqueryd_system="$system"
    if [[ $system == "windows" ]]; then
        osqueryd="$osqueryd.exe"
    elif [[ $system == "macos" ]]; then
        osqueryd="$osqueryd.app.tar.gz"
        osqueryd_system="macos-app"
    fi
    osqueryd_path="$TUF_PATH/tmp/$osqueryd"
    curl https://tuf.fleetctl.com/targets/osqueryd/$osqueryd_system/$OSQUERY_VERSION/$osqueryd --output $osqueryd_path

    major=$(echo "$OSQUERY_VERSION" | cut -d "." -f 1)
    min=$(echo "$OSQUERY_VERSION" | cut -d "." -f 2)
    ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target $osqueryd_path \
        --platform $osqueryd_system \
        --name osqueryd \
        --version $OSQUERY_VERSION -t $major.$min -t $major -t stable
    rm $osqueryd_path

    goose_value="$system"
    goarch_value="" # leave it empty to use the default for the system
    if [[ $system == "macos" ]]; then
        goose_value="darwin"
	# for all platforms except Darwin, GOARCH is hardcoded to amd64 to
	# prevent cross compilation issues when building macOS arm64 binaries
	# from Linux (CGO + libraries are required)
	goarch_value="amd64"
    fi
    orbit_target=orbit-$system
    if [[ $system == "windows" ]]; then
        orbit_target="${orbit_target}.exe"
    fi

    # compiling a macOS-arm64 binary requires CGO and a macOS computer (for
    # Apple keychain, some tables, etc), if this is the case, compile an
    # universal binary.
    if [ $system == "macos" ] && [ "$(uname -s)" = "Darwin" ]; then
       CGO_ENABLED=1 \
       CODESIGN_IDENTITY=$CODESIGN_IDENTITY \
       ORBIT_VERSION=42 \
       ORBIT_BINARY_PATH=$orbit_target \
       go run ./orbit/tools/build/build.go
    else
      GOOS=$goose_value GOARCH=$goarch_value go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=42" -o $orbit_target ./orbit/cmd/orbit
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
        if [[ -z "$MACOS_USE_PREBUILT_DESKTOP_APP_TAR_GZ" ]]; then
            FLEET_DESKTOP_VERBOSE=1 \
            FLEET_DESKTOP_VERSION=42.0.0 \
            make desktop-app-tar-gz
        fi
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target desktop.app.tar.gz \
        --platform macos \
        --name desktop \
        --version 42.0.0 -t 42.0 -t 42 -t stable
        rm desktop.app.tar.gz
    fi

    # Add Nudge application on macos (if enabled).
    if [[ $system == "macos" && -n "$NUDGE" ]]; then
        curl https://tuf.fleetctl.com/targets/nudge/macos/$NUDGE_VERSION/nudge.app.tar.gz --output nudge.app.tar.gz
        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target nudge.app.tar.gz \
            --platform macos \
            --name nudge \
            --version 42.0.0 -t 42.0 -t 42 -t stable
        rm nudge.app.tar.gz
    fi

    # Add swiftDialog on macos (if enabled).
    if [[ $system == "macos" && -n "$SWIFT_DIALOG" ]]; then
        curl https://tuf.fleetctl.com/targets/swiftDialog/macos/stable/swiftDialog.app.tar.gz --output swiftDialog.app.tar.gz

        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target swiftDialog.app.tar.gz \
            --platform macos \
            --name swiftDialog \
            --version 42.0.0 -t 42.0 -t 42 -t stable
        rm swiftDialog.app.tar.gz
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

    # Add Fleet Desktop application on linux (if enabled).
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

    # Add extensions on macos (if set).
    if [[ $system == "macos" && -n "$MACOS_TEST_EXTENSIONS" ]]; then
        for extension in ${MACOS_TEST_EXTENSIONS//,/ }
        do
            extensionName=$(basename $extension)
            extensionName=$(echo "$extensionName" | cut -d'.' -f1)
            ./build/fleetctl updates add \
                --path $TUF_PATH \
                --target $extension \
                --platform macos \
                --name "extensions/$extensionName" \
                --version 42.0.0 -t 42.0 -t 42 -t stable
        done
    fi

    # Add extensions on linux (if set).
    if [[ $system == "linux" && -n "$LINUX_TEST_EXTENSIONS" ]]; then
        for extension in ${LINUX_TEST_EXTENSIONS//,/ }
        do
            extensionName=$(basename $extension)
            extensionName=$(echo "$extensionName" | cut -d'.' -f1)
            ./build/fleetctl updates add \
                --path $TUF_PATH \
                --target $extension \
                --platform linux \
                --name "extensions/$extensionName" \
                --version 42.0.0 -t 42.0 -t 42 -t stable
        done
    fi

    # Add extensions on windows (if set).
    if [[ $system == "windows" && -n "$WINDOWS_TEST_EXTENSIONS" ]]; then
        for extension in ${WINDOWS_TEST_EXTENSIONS//,/ }
        do
            extensionName=$(basename $extension)
            extensionName=$(echo "$extensionName" | cut -d'.' -f1)
            echo "$FILE" | cut -d'.' -f2
            ./build/fleetctl updates add \
                --path $TUF_PATH \
                --target $extension \
                --platform windows \
                --name "extensions/$extensionName" \
                --version 42.0.0 -t 42.0 -t 42 -t stable
        done
    fi
done
