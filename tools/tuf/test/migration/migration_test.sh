#!/bin/bash

# Script used to test the migration from a TUF repository to a new one.
# It assumes the following:
#   - User runs the script on macOS
#   - User has a Ubuntu/Fedora and Windows 10/11 VMs.
#   - Fleet is running on the macOS host and tunneled by ngrok.
#   - `fleetctl login` was ran on the localhost Fleet instance (to be able to run `fleectl query` commands).
#   - 1.37.0 is the last version of orbit that uses the old TUF repository
#   - 1.38.0 is the new version of orbit that will use the new TUF repository.
#   - Old TUF repository directory is ./test_tuf_old and server listens on 8081 and is tunneled by ngrok (runs on the macOS host).
#   - New TUF repository directory is ./test_tuf_new and server listens on 8082 and is tunneled by ngrok (runs on the macOS host).

set -e

if [ -z "$FLEET_URL" ]; then
    echo "Missing FLEET_URL"
    exit 1
fi
if [ -z "$NO_TEAM_ENROLL_SECRET" ]; then
    echo "Missing NO_TEAM_ENROLL_SECRET"
    exit 1
fi

if [ -z "$HOSTNAMES" ]; then
    echo "Missing HOSTNAME, must be list of hostnames space-separated"
    exit 1
fi
THIS_HOSTNAME=$(hostname)
HOSTNAMES_LIST="$THIS_HOSTNAME $HOSTNAMES"
read -r -a all_hostnames <<< "$HOSTNAMES_LIST"
echo "Testing on the following hostnames:"
printf '* %s\n' "${all_hostnames[@]}"

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

prompt "Please change 'const' to 'var' in orbit/pkg/update/update.go."

echo "Uninstalling fleetd from macOS..."
sudo orbit/tools/cleanup/cleanup_macos.sh
prompt "Please manually uninstall fleetd from $HOSTNAMES."

OLD_TUF_PORT=8081
if [ -z "$OLD_TUF_URL" ]; then
	OLD_TUF_URL=http://host.docker.internal:$OLD_TUF_PORT
else
	echo "Using the provided URL '$OLD_TUF_URL' for the old TUF server"
fi
OLD_TUF_PATH=test_tuf_old
OLD_FULL_VERSION=1.37.0
OLD_MINOR_VERSION=1.37

NEW_TUF_PORT=8082
if [ -z "$NEW_TUF_URL" ]; then
	NEW_TUF_URL=http://host.docker.internal:$NEW_TUF_PORT
else
	echo "Using the provided URL '$NEW_TUF_URL' for the new TUF server"
fi
NEW_TUF_PATH=test_tuf_new
NEW_FULL_VERSION=1.38.0
NEW_MINOR_VERSION=1.38
NEW_PATCH_VERSION=1.38.1

echo "Cleaning up existing directories and file servers..."
rm -rf "$OLD_TUF_PATH"
rm -rf "$NEW_TUF_PATH"
pkill file-server || true

echo "Restoring update_channels for \"No team\" to 'stable' defaults..."
cat << EOF > upgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: stable
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f upgrade.yml

echo "Generating a TUF repository on $OLD_TUF_PATH (aka \"old\")..."
SYSTEMS="macos linux windows linux-arm64" \
TUF_PATH=$OLD_TUF_PATH \
TUF_PORT=$OLD_TUF_PORT \
FLEET_DESKTOP=1 \
./tools/tuf/test/main.sh

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

echo "Downloading and pushing latest released orbit from https://tuf.fleetctl.com to the old repository..."
curl https://tuf.fleetctl.com/targets/orbit/macos/$OLD_FULL_VERSION/orbit --output orbit-darwin
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $OLD_FULL_VERSION -t $OLD_MINOR_VERSION -t 1 -t stable
curl https://tuf.fleetctl.com/targets/orbit/linux/$OLD_FULL_VERSION/orbit --output orbit-linux-amd64
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $OLD_FULL_VERSION -t $OLD_MINOR_VERSION -t 1 -t stable
curl https://tuf.fleetctl.com/targets/orbit/linux-arm64/$OLD_FULL_VERSION/orbit --output orbit-linux-arm64
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $OLD_FULL_VERSION -t $OLD_MINOR_VERSION -t 1 -t stable
curl https://tuf.fleetctl.com/targets/orbit/windows/$OLD_FULL_VERSION/orbit.exe --output orbit.exe
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $OLD_FULL_VERSION -t $OLD_MINOR_VERSION -t 1 -t stable

echo "Building fleetd packages using old repository and old fleetctl version..."
curl -L https://github.com/fleetdm/fleet/releases/download/fleet-v4.60.0/fleetctl_v4.60.0_macos.tar.gz --output ./build/fleetctl_v4.60.0_macos.tar.gz
cd ./build
tar zxf fleetctl_v4.60.0_macos.tar.gz
cp fleetctl_v4.60.0_macos/fleetctl fleetctl-v4.60.0
cd ..
chmod +x ./build/fleetctl-v4.60.0
ROOT_KEYS1=$(./build/fleetctl-v4.60.0 updates roots --path $OLD_TUF_PATH)
declare -a pkgTypes=("pkg" "deb" "msi" "rpm")
for pkgType in "${pkgTypes[@]}"; do
    ./build/fleetctl-v4.60.0 package --type="$pkgType" \
        --enable-scripts \
        --fleet-desktop \
        --fleet-url="$FLEET_URL" \
        --enroll-secret="$NO_TEAM_ENROLL_SECRET" \
        --debug \
        --update-roots="$ROOT_KEYS1" \
        --update-url=$OLD_TUF_URL \
        --disable-open-folder \
        --disable-keystore \
        --update-interval=30s
    if [ "$pkgType" == "deb" ] || [ "$pkgType" == "rpm" ]; then
        ./build/fleetctl-v4.60.0 package --type="$pkgType" \
            --arch=arm64 \
            --enable-scripts \
            --fleet-desktop \
            --fleet-url="$FLEET_URL" \
            --enroll-secret="$NO_TEAM_ENROLL_SECRET" \
            --debug \
            --update-roots="$ROOT_KEYS1" \
            --update-url=$OLD_TUF_URL \
            --disable-open-folder \
            --disable-keystore \
            --update-interval=30s
    fi
done

# Install fleetd generated with old fleetctl and using old TUF on devices.
echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /
CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi, $CURRENT_DIR/fleet-osquery_${OLD_FULL_VERSION}_amd64.deb, $CURRENT_DIR/fleet-osquery_${OLD_FULL_VERSION}_arm64.deb, $CURRENT_DIR/fleet-osquery-${OLD_FULL_VERSION}.x86_64.rpm and $CURRENT_DIR/fleet-osquery-${OLD_FULL_VERSION}.aarch64.rpm."

echo "Generating a new TUF repository from scratch on $NEW_TUF_PATH..."
./build/fleetctl updates init --path $NEW_TUF_PATH

echo "Migrating all targets from old to new repository..."
go run ./tools/tuf/migrate/migrate.go \
    -source-repository-directory "$OLD_TUF_PATH" \
    -dest-repository-directory "$NEW_TUF_PATH"

echo "Serving new TUF repository..."
TUF_PORT=$NEW_TUF_PORT TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/run_server.sh 

echo "Building the new orbit that will perform the migration..."
ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -o orbit-darwin-amd64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build \
    -o orbit-darwin-arm64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
lipo -create orbit-darwin-amd64 orbit-darwin-arm64 -output orbit-darwin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux-amd64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -o orbit-linux-arm64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -o orbit.exe \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit

echo "Pushing new orbit to new repository on stable channel..."
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Simulating outage of the new TUF repository by killing the new TUF server..."
    # We kill the two servers and bring back the old one.
    pkill file-server || true
    TUF_PORT=$OLD_TUF_PORT TUF_PATH=$OLD_TUF_PATH ./tools/tuf/test/run_server.sh
fi

echo "Pushing new orbit to old repository!..."
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Checking version of updated orbit (to check device is responding even if TUF server is down)..."
    for host_hostname in "${all_hostnames[@]}"; do
        ORBIT_VERSION=""
        until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
            sleep 1
            ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
        done
    done

    prompt "Please check for errors in orbit logs that new TUF server is unavailable (network errors). Errors should be shown every 10s."

    echo "Bring new TUF server back but still unavailable (404s errors)."
    mkdir -p $NEW_TUF_PATH/tmp
    mv $NEW_TUF_PATH/repository/targets/* $NEW_TUF_PATH/tmp/

    TUF_PORT=$NEW_TUF_PORT TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/run_server.sh

    prompt "Please check for errors in orbit logs that new TUF server is still unavailable (404s errors). Errors should be shown every 10s."

    echo "Checking version of orbit (to check device is responding even if TUF server is down)..."
    for host_hostname in "${hostnames[@]}"; do
        ORBIT_VERSION=""
        until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
            sleep 1
            ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
        done
    done

    # We kill the two servers and bring back the old one.
    pkill file-server || true
    TUF_PORT=$OLD_TUF_PORT TUF_PATH=$OLD_TUF_PATH ./tools/tuf/test/run_server.sh
    # Restore files on the new repository.
    mv $NEW_TUF_PATH/tmp/* $NEW_TUF_PATH/repository/targets/

    if [ "$ORBIT_PATCH_IN_OLD_TUF" = "1" ]; then
        echo "Build and push a new update to orbit to old and new repository (to test patching an invalid 1.38.0 would work for customers without access to new TUF)"
        ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
        CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
            -o orbit-darwin-amd64 \
            -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
            ./orbit/cmd/orbit
        CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build \
            -o orbit-darwin-arm64 \
            -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
            ./orbit/cmd/orbit
        lipo -create orbit-darwin-amd64 orbit-darwin-arm64 -output orbit-darwin
	    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -o orbit-linux-amd64 \
            -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
            ./orbit/cmd/orbit
	    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
            -o orbit-linux-arm64 \
            -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
            ./orbit/cmd/orbit
	    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
            -o orbit.exe \
            -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
            -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
            ./orbit/cmd/orbit
        ./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
        ./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

        echo "Checking orbit has auto-updated to $NEW_PATCH_VERSION using old TUF..."
        for host_hostname in "${hostnames[@]}"; do
            ORBIT_VERSION=""
            until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
            sleep 1
            ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
            done
        done

        # Now the next patch version will be 1.38.2.
        NEW_FULL_VERSION=1.38.1
        NEW_PATCH_VERSION=1.38.2
    fi

    echo "Restoring new TUF repository..."
    TUF_PORT=$NEW_TUF_PORT TUF_PATH=$NEW_TUF_PATH ./tools/tuf/test/run_server.sh

    prompt "Please check that devices have restarted and started communicating with the new TUF (now that it's available)"
fi

echo "Checking version of updated orbit..."
THIS_HOSTNAME=$(hostname)
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Restarting fleetd on the macOS host..."
sudo launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist && sudo launchctl load /Library/LaunchDaemons/com.fleetdm.orbit.plist

prompt "Please restart fleetd on the Linux and Windows host."

echo "Checking version of updated orbit..."
THIS_HOSTNAME=$(hostname)
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Building and pushing a new update to orbit on the new repository (to test upgrades are working)..."
ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -o orbit-darwin-amd64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build \
    -o orbit-darwin-arm64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
lipo -create orbit-darwin-amd64 orbit-darwin-arm64 -output orbit-darwin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux-amd64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -o orbit-linux-arm64 \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -o orbit.exe \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-amd64 --platform linux --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux-arm64 --platform linux-arm64 --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

echo "Waiting until update happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

prompt "Check that your orbit is on 1.38.2 on all your devices"
echo "Downgrading to $OLD_FULL_VERSION..."
cat << EOF > downgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: '$OLD_FULL_VERSION'
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f downgrade.yml

echo "Waiting until downgrade happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$OLD_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Restoring to latest orbit version..."
cat << EOF > upgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: stable
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f upgrade.yml

echo "Waiting until upgrade happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Building fleetd packages using old repository and old fleetctl version that should auto-update to new orbit that talks to new repository..."
for pkgType in "${pkgTypes[@]}"; do
    ./build/fleetctl-v4.60.0 package --type="$pkgType" \
        --enable-scripts \
        --fleet-desktop \
        --fleet-url="$FLEET_URL" \
        --enroll-secret="$NO_TEAM_ENROLL_SECRET" \
        --debug \
        --update-roots="$ROOT_KEYS1" \
        --update-url=$OLD_TUF_URL \
        --disable-open-folder \
        --disable-keystore \
        --update-interval=30s
done

echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /

CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi, $CURRENT_DIR/fleet-osquery_${NEW_FULL_VERSION}_amd64.deb, $CURRENT_DIR/fleet-osquery_${NEW_FULL_VERSION}_arm64.deb, $CURRENT_DIR/fleet-osquery-${NEW_FULL_VERSION}.x86_64.rpm and $CURRENT_DIR/fleet-osquery-${NEW_FULL_VERSION}.aarch64.rpm."

echo "Waiting until installation and auto-update to new repository happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Downgrading to $OLD_FULL_VERSION..."
cat << EOF > downgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: '$OLD_FULL_VERSION'
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f downgrade.yml

echo "Waiting until downgrade happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$OLD_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Restoring to latest orbit version..."
cat << EOF > upgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: stable
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f upgrade.yml

echo "Waiting until upgrade happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done


echo "Building fleetd packages using new repository and new fleetctl version..."

CGO_ENABLED=0 go build \
    -o ./build/fleetctl \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL" \
    ./cmd/fleetctl

for pkgType in "${pkgTypes[@]}"; do
    ./build/fleetctl package --type="$pkgType" \
        --enable-scripts \
        --fleet-desktop \
        --fleet-url="$FLEET_URL" \
        --enroll-secret="$NO_TEAM_ENROLL_SECRET" \
        --debug \
        --disable-open-folder \
        --disable-keystore \
        --update-interval=30s
done

echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /

CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi, $CURRENT_DIR/fleet-osquery_${NEW_PATCH_VERSION}_amd64.deb, $CURRENT_DIR/fleet-osquery_${NEW_PATCH_VERSION}_arm64.deb, $CURRENT_DIR/fleet-osquery-${NEW_PATCH_VERSION}.x86_64.rpm and $CURRENT_DIR/fleet-osquery-${NEW_PATCH_VERSION}.aarch64.rpm."

echo "Waiting until installation and auto-update to new repository happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_PATCH_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

cat << EOF > downgrade.yml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        pack_delimiter: /
        distributed_plugin: tls
        disable_distributed: false
        logger_tls_endpoint: /api/v1/osquery/log
        distributed_interval: 10
        distributed_tls_max_attempts: 3
        distributed_denylist_duration: 10
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
    update_channels:
      orbit: '$OLD_FULL_VERSION'
      desktop: stable
      osqueryd: stable
EOF
fleetctl apply -f downgrade.yml

echo "Waiting until downgrade happens..."
for host_hostname in "${all_hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$OLD_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Migration testing completed."
