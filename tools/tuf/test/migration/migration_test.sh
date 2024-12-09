#!/bin/bash

# Script used to test the migration from a TUF repository to a new one.
# It assumes the following:
#   - User runs the script on macOS
#   - User has a Ubuntu 22.04 and a Windows 10/11 VM (running on the same macOS host script runs on).
#   - Fleet is running on the macOS host.
#   - `fleetctl login` was ran on the localhost Fleet instance (to be able to run `fleectl query` commands).
#   - host.docker.internal points to localhost on the macOS host.
#   - host.docker.internal points to the macOS host on the two VMs (/etc/hosts on Ubuntu and C:\Windows\System32\Drivers\etc\hosts on Windows).
#   - 1.36.0 is the last version of orbit that uses the old TUF repository
#   - 1.37.0 is the new version of orbit that will use the new TUF repository.
#   - Old TUF repository directory is ./test_tuf_old and server listens on 8081 (runs on the macOS host).
#   - New TUF repository directory is ./test_tuf_new and server listens on 8082 (runs on the macOS host).

set -e

if [ -z "$FLEET_URL" ]; then
    echo "Missing FLEET_URL"
    exit 1
fi
if [ -z "$NO_TEAM_ENROLL_SECRET" ]; then
    echo "Missing NO_TEAM_ENROLL_SECRET"
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
OLD_FULL_VERSION=1.36.0
OLD_MINOR_VERSION=1.36

NEW_TUF_PORT=8082
NEW_TUF_URL=http://host.docker.internal:$NEW_TUF_PORT
NEW_TUF_PATH=test_tuf_new
NEW_FULL_VERSION=1.37.0
NEW_MINOR_VERSION=1.37
NEW_PATCH_VERSION=1.37.1

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
SYSTEMS="macos linux windows" \
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
curl https://tuf.fleetctl.com/targets/orbit/linux/$OLD_FULL_VERSION/orbit --output orbit-linux
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux --platform linux --name orbit --version $OLD_FULL_VERSION -t $OLD_MINOR_VERSION -t 1 -t stable
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
declare -a pkgTypes=("pkg" "deb" "msi")
for pkgType in "${pkgTypes[@]}"; do
    ./build/fleetctl-v4.60.0 package --type="$pkgType" \
        --enable-scripts \
        --fleet-desktop \
        --fleet-url="$FLEET_URL" \
        --enroll-secret="$NO_TEAM_ENROLL_SECRET" \
        --fleet-certificate=./tools/osquery/fleet.crt \
        --debug \
        --update-roots="$ROOT_KEYS1" \
        --update-url=$OLD_TUF_URL \
        --disable-open-folder \
        --disable-keystore \
        --update-interval=30s
done

# Install fleetd generated with old fleetctl and using old TUF on devices.
echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /
CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi and $CURRENT_DIR/fleet-osquery_${OLD_FULL_VERSION}_amd64.deb."

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
    -o orbit-darwin \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_FULL_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux \
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
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux --platform linux --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Simulating outage of the new TUF repository..."
    mkdir $NEW_TUF_PATH/tmp
    mv $NEW_TUF_PATH/repository/*.json $NEW_TUF_PATH/tmp/
fi

echo "Pushing new orbit to old repository!..."
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-darwin --platform macos --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit-linux --platform linux --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $OLD_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_FULL_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

if [ "$SIMULATE_NEW_TUF_OUTAGE" = "1" ]; then
    echo "Checking version of updated orbit (to check device is responding even if TUF server is down)..."
    THIS_HOSTNAME=$(hostname)
    declare -a hostnames=("$THIS_HOSTNAME" "$WINDOWS_HOST_HOSTNAME" "$LINUX_HOST_HOSTNAME")
    for host_hostname in "${hostnames[@]}"; do
        ORBIT_VERSION=""
        until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
            sleep 1
            ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
        done
    done

    echo "Waiting until migration is probed and expected to fail on the macOS host (greping logs)..."
    until sudo grep "update metadata: client update: tuf: " /var/log/orbit/orbit.stderr.log; do
        sleep 1
    done
    prompt "Please check for \"update metadata: client update: tuf: \" in the logs on the Linux and Windows host. And make sure devices are working as expected even if new TUF is down, by running live queries."

    echo "Restoring new TUF repository..."
    mv $NEW_TUF_PATH/tmp/*.json $NEW_TUF_PATH/repository/
fi

echo "Checking version of updated orbit..."
THIS_HOSTNAME=$(hostname)
declare -a hostnames=("$THIS_HOSTNAME" "$WINDOWS_HOST_HOSTNAME" "$LINUX_HOST_HOSTNAME")
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$NEW_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Building and pushing a new update to orbit on the new repository (to test upgrades are working)..."
ROOT_KEYS2=$(./build/fleetctl updates roots --path $NEW_TUF_PATH)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -o orbit-darwin \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$NEW_PATCH_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.DefaultURL=$NEW_TUF_URL \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.defaultRootMetadata=$ROOT_KEYS2 \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/update.OldFleetTUFURL=$OLD_TUF_URL" \
    ./orbit/cmd/orbit
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o orbit-linux \
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
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit-linux --platform linux --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable
./build/fleetctl updates add --path $NEW_TUF_PATH --target ./orbit.exe --platform windows --name orbit --version $NEW_PATCH_VERSION -t $NEW_MINOR_VERSION -t 1 -t stable

echo "Waiting until update happens..."
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
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
        --fleet-certificate=./tools/osquery/fleet.crt \
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
prompt "Please install $CURRENT_DIR/fleet-osquery.msi and $CURRENT_DIR/fleet-osquery_${NEW_FULL_VERSION}_amd64.deb."

echo "Waiting until installation and auto-update to new repository happens..."
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
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
        --fleet-certificate=./tools/osquery/fleet.crt \
        --debug \
        --disable-open-folder \
        --disable-keystore \
        --update-interval=30s
done

echo "Installing fleetd package on macOS..."
sudo installer -pkg fleet-osquery.pkg -verbose -target /

CURRENT_DIR=$(pwd)
prompt "Please install $CURRENT_DIR/fleet-osquery.msi and $CURRENT_DIR/fleet-osquery_${NEW_PATCH_VERSION}_amd64.deb."

echo "Waiting until installation and auto-update to new repository happens..."
for host_hostname in "${hostnames[@]}"; do
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
for host_hostname in "${hostnames[@]}"; do
    ORBIT_VERSION=""
    until [ "$ORBIT_VERSION" = "\"$OLD_FULL_VERSION\"" ]; do
        sleep 1
        ORBIT_VERSION=$(fleetctl query --hosts "$host_hostname" --exit --query 'SELECT * FROM orbit_info;' 2>/dev/null | jq '.rows[0].version')
    done
done

echo "Migration testing completed."