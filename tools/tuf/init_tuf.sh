#!/bin/bash

set -e

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

export TUF_PATH=test_tuf

if [ -n "$GENERATE_PKGS" ] && [ -z "$ENROLL_SECRET" ]; then
  echo "Error: To generate package you must set ENROLL_SECRET variable."
  exit 1
fi

make fleetctl

function create_repository() {
  ./build/fleetctl updates init --path $TUF_PATH

  for system in macos linux windows; do
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

  done

  # Generate and add osqueryd .app bundle for macos-app.
  osqueryd_path=$TUF_PATH/tmp/osqueryd.app.tar.gz
  make osqueryd-app-tar-gz version=5.2.2 out-path=$(dirname $osqueryd_path)
  osquery_path=$TUF_PATH/path/osqueryd.app.tar.gz
  ./build/fleetctl updates add \
    --path $TUF_PATH \
    --target $osqueryd_path \
    --platform macos-app \
    --name osqueryd \
    --version 42.0.0 -t 42.0 -t 42 -t stable
  rm $osqueryd_path
}

if [ ! -d "$TUF_PATH/repository" ]; then
  mkdir -p $TUF_PATH
  mkdir -p $TUF_PATH/tmp
  create_repository
fi

root_keys=$(./build/fleetctl updates roots --path $TUF_PATH)

echo "#########"
echo "To generate packages set the following options in 'fleetctl package':"
echo "--update-roots='$root_keys' --update-url=http://localhost:8081"
echo "#########"

echo "Running TUF server..."
go run ./tools/file-server 8081 "${TUF_PATH}/repository" &
SERVER_PID=$!

if [ -n "$GENERATE_PKGS" ]; then
  sleep 5

  # Change these values accordingly
  PKG_HOSTNAME=localhost
  DEB_HOSTNAME=172.16.132.1
  RPM_HOSTNAME=172.16.132.1
  MSI_HOSTNAME=172.16.132.1

  echo "Generating pkg..."
  ./build/fleetctl package \
    --type=pkg \
    ${FLEET_DESKTOP:+--fleet-desktop} \
    --fleet-url=https://$PKG_HOSTNAME:8080 \
    --enroll-secret=$ENROLL_SECRET \
    --insecure \
    --debug \
    --update-roots="$root_keys" \
    --update-url=http://$PKG_HOSTNAME:8081

  echo "Generating deb..."
  ./build/fleetctl package \
    --type=deb \
    --fleet-url=https://$DEB_HOSTNAME:8080 \
    --enroll-secret=$ENROLL_SECRET \
    --insecure \
    --debug \
    --update-roots="$root_keys" \
    --update-url=http://$DEB_HOSTNAME:8081

  echo "Generating rpm..."
  ./build/fleetctl package \
    --type=rpm \
    --fleet-url=https://$RPM_HOSTNAME:8080 \
    --enroll-secret=$ENROLL_SECRET \
    --insecure \
    --debug \
    --update-roots="$root_keys" \
    --update-url=http://$RPM_HOSTNAME:8081

  echo "Generating msi..."
  ./build/fleetctl package \
    --type=msi \
    --fleet-url=https://$MSI_HOSTNAME:8080 \
    --enroll-secret=$ENROLL_SECRET \
    --insecure \
    --debug \
    --update-roots="$root_keys" \
    --update-url=http://$MSI_HOSTNAME:8081

  echo "Packages generated"
fi

wait $SERVER_PID
