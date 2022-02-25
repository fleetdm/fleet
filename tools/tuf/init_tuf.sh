#!/bin/bash

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

export TUF_PATH=test_tuf

function create_repository() {
  ./build/fleetctl updates init --path $TUF_PATH

  for system in macos; do
    osqueryd="osqueryd"
    ./build/fleetctl updates add \
      --path $TUF_PATH \
      --target $osqueryd \
      --platform $system \
      --name osqueryd \
      --version 42.0.0 -t 42.0 -t 42 -t stable
    goose_value="$system"
    if [[ $system == "macos" ]]; then
      goose_value="darwin"
    fi
    orbit_target=orbit-$system
    if [[ $system == "windows" ]]; then
      orbit_target="${orbit_target}.exe"
    fi
    # Compile the latest version of orbit from source.
    GOOS=$goose_value go build -o $orbit_target ./orbit/cmd/orbit
    ./build/fleetctl updates add \
      --path $TUF_PATH \
      --target $orbit_target \
      --platform $system \
      --name orbit \
      --version 42.0.0 -t 42.0 -t 42 -t stable
  done
}


if [ ! -d "$TUF_PATH/repository" ]; then
  mkdir -p $TUF_PATH
  create_repository
fi

root_keyid=$(cat $TUF_PATH/repository/root.json | jq '.signed.roles.root.keyids[0]')
root_key=$(cat $TUF_PATH/repository/root.json | jq '.signed.keys.'"${root_keyid}"'' | jq -c)
root_keys="[$root_key]"

echo "#########"
echo "Set the following options in 'fleetctl package':"
echo "--update-roots='$root_keys' --update-url=http://localhost:8081"
echo "#########"

echo "Running TUF server..."
go run ./tools/file-server 8081 $TUF_PATH/repository
