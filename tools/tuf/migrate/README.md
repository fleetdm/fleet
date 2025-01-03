# migrate

This tool will be used to migrate all current targets (except unused ones) from https://tuf.fleetctl.com to https://updates.fleetdm.com.

Usage:
```sh
# The tool requires the 'targets', 'snapshot' and 'timestamp' roles of the new repository.
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

#
# It assumes the following:
# - https://tuf.fleetctl.com was fully fetched into -source-repository-directory.
# - https://updates.fleetdm.com was fully fetched into -dest-repository-directory.
#
# Migration may take several minutes due to sha512 verification after targets are
# added to the new repository.
go run ./tools/tuf/migrate/migrate.go \
    -source-repository-directory ./source-tuf-directory \
    -dest-repository-directory ./dest-tuf-directory
```
