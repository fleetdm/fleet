#!/bin/bash

system=$1
target_name=$2
target_path=$3
major_version=$4

if [ -z "$TUF_PATH" ]; then
  TUF_PATH=test_tuf
fi
export TUF_PATH

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

./build/fleetctl updates add --path $TUF_PATH --target $target_path --platform $system --name $target_name --version $major_version.0.0 -t $major_version.0 -t $major_version -t stable