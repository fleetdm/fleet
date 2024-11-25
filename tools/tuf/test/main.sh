#!/bin/bash

set -ex

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

if [ -n "$KEY_EXPIRATION_DURATION" ]; then
  export EXTRA_FLEETCTL_LDFLAGS="$EXTRA_FLEETCTL_LDFLAGS -X github.com/fleetdm/fleet/v4/ee/fleetctl.keyExpirationDuration=$KEY_EXPIRATION_DURATION"
fi
if [ -n "$SNAPSHOT_EXPIRATION_DURATION" ]; then
  export EXTRA_FLEETCTL_LDFLAGS="$EXTRA_FLEETCTL_LDFLAGS -X github.com/fleetdm/fleet/v4/ee/fleetctl.snapshotExpirationDuration=$SNAPSHOT_EXPIRATION_DURATION"
fi
if [ -n "$TARGETS_EXPIRATION_DURATION" ]; then
  export EXTRA_FLEETCTL_LDFLAGS="$EXTRA_FLEETCTL_LDFLAGS -X github.com/fleetdm/fleet/v4/ee/fleetctl.targetsExpirationDuration=$TARGETS_EXPIRATION_DURATION"
fi
if [ -n "$TIMESTAMP_EXPIRATION_DURATION" ]; then
  export EXTRA_FLEETCTL_LDFLAGS="$EXTRA_FLEETCTL_LDFLAGS -X github.com/fleetdm/fleet/v4/ee/fleetctl.timestampExpirationDuration=$TIMESTAMP_EXPIRATION_DURATION"
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

if [ -n "$GENERATE_PKG" ] || [ -n "$GENERATE_DEB" ] || [ -n "$GENERATE_RPM" ] || [ -n "$GENERATE_MSI" ] || [ -n "$GENERATE_DEB_ARM64" ] || [ -n "$GENERATE_RPM_ARM64" ]; then
    bash ./tools/tuf/test/gen_pkgs.sh
fi
