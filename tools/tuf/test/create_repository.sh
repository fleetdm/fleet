#!/bin/bash

set -xe

# This script initializes a test Fleet TUF repository.
# All targets are created with version set to YY.MM.XXXXX, e.g. 25.5.56178
# (patch version is a 16-bit number made from day, hour and minute).

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
    set +x
    echo "Do you want to remove the existing $TUF_PATH directory?"
    echo "Type 'yes/no' to continue... "
    while read -r word;
    do
        if [[ "$word" == "yes" ]]; then
            rm -rf "$TUF_PATH"
            break
        elif [[ "$word" == "no" ]]; then
            break
        fi
    done
    set -x
fi

SYSTEMS=${SYSTEMS:-macos linux linux-arm64 windows windows-arm64}

echo "Generating packages for $SYSTEMS"

NUDGE_VERSION=stable
ESCROW_BUDDY_PKG_VERSION=1.0.0

if [[ -z "$OSQUERY_VERSION" ]]; then
    OSQUERY_VERSION=5.16.0
fi

mkdir -p $TUF_PATH/tmp

./build/fleetctl updates init --path $TUF_PATH

source ./tools/tuf/test/load_orbit_version_vars.sh

for system in $SYSTEMS; do

    # Use latest stable version of osqueryd from our TUF server.
    osqueryd="osqueryd"
    osqueryd_system="$system"
    if [[ $system == "windows" ]]; then
        osqueryd="$osqueryd.exe"
    elif [[ $system == "windows-arm64" ]]; then
        osqueryd="$osqueryd.exe"
        osqueryd_system="windows-arm64"
    elif [[ $system == "macos" ]]; then
        osqueryd="$osqueryd.app.tar.gz"
        osqueryd_system="macos-app"
    elif [[ $system == "linux-arm64" ]]; then
        osqueryd_system="linux-arm64"
    fi

    if [[ $system == "linux-arm64" ]]; then
        osqueryd_path="$TUF_PATH/tmp/${osqueryd}-arm64"
    else
        osqueryd_path="$TUF_PATH/tmp/$osqueryd"
    fi
    curl https://updates.fleetdm.com/targets/osqueryd/$osqueryd_system/$OSQUERY_VERSION/$osqueryd --output $osqueryd_path

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
    goarch_value=${GOARCH:-}
    if [[ $system == "macos" ]]; then
        goose_value="darwin"
    fi
    if [[ $system == "linux" ]]; then
        goarch_value="amd64"
    fi
    if [[ $system == "windows" ]]; then
        goarch_value="amd64"
    fi
    if [[ $system == "windows-arm64" ]]; then
        goose_value="windows"
        goarch_value="arm64"
    fi
    if [[ $system == "linux-arm64" ]]; then
        goose_value="linux"
        goarch_value="arm64"
    fi
    orbit_target=orbit-$system
    if [[ $system == "windows" ]] || [[ $system == "windows-arm64" ]]; then
        orbit_target="${orbit_target}.exe"
    fi

    # compiling a macOS-arm64 binary requires CGO and a macOS computer (for
    # Apple keychain, some tables, etc), if this is the case, compile an
    # universal binary.
    #
    if [ $system == "macos" ] && [ "$(uname -s)" = "Darwin" ] && [ "$(uname -m)" = "arm64" ]; then
       CGO_ENABLED=1 \
       CODESIGN_IDENTITY=$CODESIGN_IDENTITY \
       ORBIT_VERSION=$ORBIT_VERSION \
       ORBIT_COMMIT=$ORBIT_COMMIT \
       ORBIT_BINARY_PATH=$orbit_target \
       go run ./orbit/tools/build/build.go
    else
      race_value=false
      # Enable race on macOS Intel at least.
      #
      # For cross-compiling to Windows with `-race` we need CGO_ENABLED=1 but we cannot
      # do cross-compilation with CGO_ENABLED=1.
      if [ "$goose_value" = "darwin" ] && [ "$(uname -s)" = "Darwin" ] && [ "$(uname -m)" = "x86_64" ]; then
        race_value=true
      fi
      # NOTE(lucas): Cross-compiling orbit for arm64 from Intel macOS currently fails (CGO error),
      # thus on Intel we do not build an universal binary.
      CGO_ENABLED=0 \
      GOOS=$goose_value \
      GOARCH=$goarch_value \
      go build \
      -race=$race_value \
      -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION \
        -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" \
      -o $orbit_target ./orbit/cmd/orbit
    fi

    ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target $orbit_target \
        --platform $system \
        --name orbit \
        --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
    rm $orbit_target

    # Add Fleet Desktop application on macos (if enabled).
    if [[ $system == "macos" && -n "$FLEET_DESKTOP" ]]; then
        if [[ -z "$MACOS_USE_PREBUILT_DESKTOP_APP_TAR_GZ" ]]; then
            FLEET_DESKTOP_VERBOSE=1 \
            FLEET_DESKTOP_VERSION=$ORBIT_VERSION \
            make desktop-app-tar-gz
        fi
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target desktop.app.tar.gz \
        --platform macos \
        --name desktop \
        --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        if [[ -z "$MACOS_USE_PREBUILT_DESKTOP_APP_TAR_GZ" ]]; then
            rm desktop.app.tar.gz
        fi
    fi

    # Add Nudge application on macos (if enabled).
    if [[ $system == "macos" && -n "$NUDGE" ]]; then
        curl https://updates.fleetdm.com/targets/nudge/macos/$NUDGE_VERSION/nudge.app.tar.gz --output nudge.app.tar.gz
        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target nudge.app.tar.gz \
            --platform macos \
            --name nudge \
            --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm nudge.app.tar.gz
    fi

    # Add swiftDialog on macos (if enabled).
    if [[ $system == "macos" && -n "$SWIFT_DIALOG" ]]; then
        curl https://updates.fleetdm.com/targets/swiftDialog/macos/stable/swiftDialog.app.tar.gz --output swiftDialog.app.tar.gz

        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target swiftDialog.app.tar.gz \
            --platform macos \
            --name swiftDialog \
            --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm swiftDialog.app.tar.gz
    fi

    # Add Escrow Buddy on macos (if enabled).
    if [[ $system == "macos" && -n "$ESCROW_BUDDY" ]]; then
	make escrow-buddy-pkg version=$ESCROW_BUDDY_PKG_VERSION out-path=.

        ./build/fleetctl updates add \
            --path $TUF_PATH \
            --target escrowBuddy.pkg \
            --platform macos \
            --name escrowBuddy \
            --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm escrowBuddy.pkg
    fi


    # Add Fleet Desktop application on indows (if enabled).
    if [[ $system == "windows" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=$ORBIT_VERSION \
        make desktop-windows
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target fleet-desktop.exe \
        --platform windows \
        --name desktop \
        --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm fleet-desktop.exe
    fi

    # Add Fleet Desktop application on windows-arm64 (if enabled).
    if [[ $system == "windows-arm64" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=$ORBIT_VERSION \
        make desktop-windows-arm64
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target fleet-desktop.exe \
        --platform windows-arm64 \
        --name desktop \
        --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm fleet-desktop.exe
    fi

    # Add Fleet Desktop application on linux (if enabled).
    if [[ $system == "linux" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=$ORBIT_VERSION \
        make desktop-linux
        ./build/fleetctl updates add \
        --path $TUF_PATH \
        --target desktop.tar.gz \
        --platform linux \
        --name desktop \
        --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        rm desktop.tar.gz
    fi

    # Add Fleet Desktop application on linux-arm64 (if enabled).
    if [[ $system == "linux-arm64" && -n "$FLEET_DESKTOP" ]]; then
        FLEET_DESKTOP_VERSION=$ORBIT_VERSION \
        make desktop-linux-arm64
        ./build/fleetctl updates add \
                         --path $TUF_PATH \
                         --target desktop.tar.gz \
                         --platform linux-arm64 \
                         --name desktop \
                         --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
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
                --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
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
                --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        done
    fi

    # Add extensions on linux (if set).
    if [[ $system == "linux-arm64" && -n "$LINUX_TEST_EXTENSIONS" ]]; then
        for extension in ${LINUX_TEST_EXTENSIONS//,/ }
        do
            extensionName=$(basename $extension)
            extensionName=$(echo "$extensionName" | cut -d'.' -f1)
            ./build/fleetctl updates add \
                             --path $TUF_PATH \
                             --target $extension \
                             --platform linux-arm64 \
                             --name "extensions/$extensionName" \
                             --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
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
                --version $ORBIT_VERSION -t $ORBIT_MAJOR.$ORBIT_MINOR -t $ORBIT_MAJOR -t stable
        done
    fi
done
