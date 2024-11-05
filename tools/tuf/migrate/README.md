# migrate

This tool will be used to migrate all current targets (except `orbit` and unused ones) from https://tuf.fleetctl.com to https://updates.fleetdm.com.

Usage:
```sh
# The tool requires the 'targets', 'snapshot' and 'timestamp' roles.
export FLEET_TARGETS_PASSPHRASE=p4ssphr4s3
export FLEET_SNAPSHOT_PASSPHRASE=p4ssphr4s3
export FLEET_TIMESTAMP_PASSPHRASE=p4ssphr4s3

go run ./tools/tuf/migrate/migrate.go \
    -source-repository-directory ./source-tuf-directory \
    -dest-repository-directory ./dest-tuf-directory
```
