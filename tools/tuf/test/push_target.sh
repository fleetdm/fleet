#!/bin/bash

system=$1
target_name=$2
target_path=$3
version=$4

if [ -z "$TUF_PATH" ]; then
  TUF_PATH=test_tuf
fi
export TUF_PATH

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

major=$(echo $version | cut -d. -f1)
minor=$(echo $version | cut -d. -f2)

./build/fleetctl updates add --path $TUF_PATH --target $target_path --platform $system --name $target_name --version $version -t "$major.$minor" -t "$major" -t stable