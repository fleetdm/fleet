#!/bin/bash

set -e

if [ -z "$FLEET_URL" ]; then
    echo "Missing FLEET_URL"
    exit 1
fi
if [ -z "$ENROLL_SECRET" ]; then
    echo "Missing ENROLL_SECRET"
    exit 1
fi

if [ -z "$WINDOWS_HOST_HOSTNAME" ]; then
    echo "Missing WINDOWS_HOST_HOSTNAME"
    exit 1
fi

if [ -z "$LINUX_HOST_HOSTNAME" ]; then
    echo "Missing LINUX_HOST_HOSTNAME"
    exit 1
fi

prompt () {
    printf "%s\n" "$1"
    printf "Type 'yes' to continue... "
    while read -r word;
    do
        if [[ "$word" == "yes" ]]; then
            printf "\n"
            return
        fi
    done
}

echo "Uinstalling fleetd from macOS..."
sudo orbit/tools/cleanup/cleanup_macos.sh
prompt "Please manually uninstall fleetd from $WINDOWS_HOST_HOSTNAME and $LINUX_HOST_HOSTNAME."

OLD_TUF_PORT=8081
OLD_TUF_URL=http://host.docker.internal:$OLD_TUF_PORT
OLD_TUF_PATH=test_tuf_old

NEW_TUF_PORT=8082
NEW_TUF_URL=http://host.docker.internal:$NEW_TUF_PORT
NEW_TUF_PATH=test_tuf_new

echo "Cleaning up existing directories and file servers..."
rm -rf "$OLD_TUF_PATH"
rm -rf "$NEW_TUF_PATH"
pkill file-server || true

echo "Generating a TUF repository on $OLD_TUF_PATH (\"old\")..."
SYSTEMS="macos linux windows" \
TUF_PATH=$OLD_TUF_PATH \
TUF_PORT=$OLD_TUF_PORT \
FLEET_DESKTOP=1 \
./tools/tuf/test/main.sh

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

echo "Downloading and pushing latest released orbit from https://tuf.fleetctl.com..."
curl https://tuf.fleetctl.com/targets/orbit/macos/1.35.0/orbit --output orbit-darwin
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version 1.35.0 -t 1.35 -t 1 -t stable
curl https://tuf.fleetctl.com/targets/orbit/linux/1.35.0/orbit --output orbit-linux
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux --platform linux --name orbit --version 1.35.0 -t 1.35 -t 1 -t stable
curl https://tuf.fleetctl.com/targets/orbit/windows/1.35.0/orbit.exe --output orbit.exe
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version 1.35.0 -t 1.35 -t 1 -t stable

echo "Building fleetd packages using old repository..."
ROOT_KEYS1=$(./build/fleetctl updates roots --path $OLD_TUF_PATH)
declare -a pkgTypes=("pkg" "deb" "msi")
for pkgType in "${pkgTypes[@]}"; do
    fleetctl package --type="$pkgType" \
        --enable-scripts \
        --fleet-desktop \
        --fleet-url="$FLEET_URL" \
        --enroll-secret="$ENROLL_SECRET" \
        --fleet-certificate=./tools/osquery/fleet.crt \
        --debug \
        --update-roots="$ROOT_KEYS1" \
        --update-url=$OLD_TUF_URL \
        --disable-open-folder \
        --update-interval=30s
done

echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /

CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi and $CURRENT_DIR/fleet-osquery_1.35.0_amd64.deb."

echo "Generating a new TUF repository from scratch on $NEW_TUF_PATH..."
./build/fleetctl updates init --path $NEW_TUF_PATH

echo "Migrating all targets from old to new repository (except \"orbit\")..."
go run ./tools/tuf/migrate/migrate.go \
    -source-repository-directory "$OLD_TUF_PATH" \
    -dest-repository-directory "$NEW_TUF_PATH"

echo "Serving new TUF repository..."
TUF_PORT=$NEW_TUF_PORT TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/run_server.sh 

echo "Building the new orbit that will perform the migration..."
ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -o orbit-darwin \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.36.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.36.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -o orbit.exe \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.36.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit

echo "Pushing orbit to new repository..."
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh macos orbit ./orbit-darwin 42
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh linux orbit ./orbit-linux 42
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh windows orbit ./orbit.exe 42

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Simulating outage of the new TUF repository..."
    mkdir $NEW_TUF_PATH/tmp
    mv $NEW_TUF_PATH/repository/*.json $NEW_TUF_PATH/tmp/
fi

echo "Pushing orbit to old repository..."
TUF_PATH=$OLD_TUF_PATH ./tools/tuf/test/push_target.sh macos orbit ./orbit-darwin 42
TUF_PATH=$OLD_TUF_PATH ./tools/tuf/test/push_target.sh linux orbit ./orbit-linux 42
TUF_PATH=$OLD_TUF_PATH ./tools/tuf/test/push_target.sh windows orbit ./orbit.exe 42

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Waiting until migration is probed and expected to fail on the macOS host (greping logs)..."
    until sudo grep "failed to probe TUF migration" /var/log/orbit/orbit.stderr.log; do
        sleep 1
    done
    prompt "Please check for \"failed to probe TUF migration\" in the logs on the Linux and Windows host. And make sure devices are working as expected."

    echo "Restoring new TUF repository..."
    mv $NEW_TUF_PATH/tmp/*.json $NEW_TUF_PATH/repository/
fi

echo "Waiting until migration happens on the macOS host (greping logs)..."
until sudo grep "migration to new TUF repository completed" /var/log/orbit/orbit.stderr.log; do
    sleep 1
done
prompt "Please check for \"migration to new TUF repository completed\" in the logs on the Linux and Windows host."

echo "Restarting fleetd on the macOS host..."
sudo launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist && sudo launchctl load /Library/LaunchDaemons/com.fleetdm.orbit.plist

prompt "Please restart fleetd on the Linux and Windows host."

echo "Waiting until migration happens (greping logs)..."
until sudo grep "nothing to do, already migrated" /var/log/orbit/orbit.stderr.log; do
    sleep 1
done
prompt "Please check for \"nothing to do, already migrated\" in the logs on the Linux and Windows host."

echo "Checking version of updated orbit..."
THIS_HOSTNAME=$(hostname)
declare -a hostnames=("$THIS_HOSTNAME" "$WINDOWS_HOST_HOSTNAME" "$LINUX_HOST_HOSTNAME")
for host_hostname in "${hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"1.36.0\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Building and pushing a new update to orbit on the new repository (to test upgrades are working)..."
ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -o orbit-darwin \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.37.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.37.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -o orbit.exe \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=1.37.0 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.oldFleetTUFRootMetadata=$ROOT_KEYS1" \
    ./orbit/cmd/orbit
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh macos orbit ./orbit-darwin 42
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh linux orbit ./orbit-linux 42
TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/push_target.sh windows orbit ./orbit.exe 42

echo "Waiting until update happens..."
declare -a hostnames=("$THIS_HOSTNAME" "$WINDOWS_HOST_HOSTNAME" "$LINUX_HOST_HOSTNAME")
for host_hostname in "${hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"1.37.0\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Migration testing completed."