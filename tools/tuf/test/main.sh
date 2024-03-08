#!/bin/bash

set -e

export FLEET_ROOT_PASSPHRASE=p4ssphr4s3
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3
export TUF_PATH=test_tuf
export NUDGE=1

if ( [ -n "$GENERATE_PKG" ] || [ -n "$GENERATE_DEB" ] || [ -n "$GENERATE_RPM" ] || [ -n "$GENERATE_MSI" ] ) && [ -z "$ENROLL_SECRET" ]; then
  echo "Error: To generate packages you must set ENROLL_SECRET variable."
  exit 1
fi

make fleetctl
./tools/tuf/test/create_repository.sh

export ROOT_KEYS=$(./build/fleetctl updates roots --path $TUF_PATH)

echo "#########"
echo "To generate packages set the following options in 'fleetctl package':"
echo "--update-roots='$ROOT_KEYS' --update-url=http://localhost:8081"
echo "You can also pass the above flags to 'fleetctl preview'."
echo "#########"

if [ -z "$SKIP_SERVER" ]; then
    ./tools/tuf/test/run_server.sh
fi

if [ -n "$GENERATE_PKG" ] || [ -n "$GENERATE_DEB" ] || [ -n "$GENERATE_RPM" ] || [ -n "$GENERATE_MSI" ]; then
  ./tools/tuf/test/gen_pkgs.sh
fi
