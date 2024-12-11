# `migration_test.sh`

This script is used to test the migration from one local TUF repository to a new local TUF repository (with new roots).

> Currently supports running on macOS only.

The script is interactive and assumes the user will use a Windows and Ubuntu VM to install fleetd and test the changes on those platforms too.

Usage:
```sh
FLEET_URL=https://host.docker.internal:8080 \
NO_TEAM_ENROLL_SECRET=... \
WINDOWS_HOST_HOSTNAME=DESKTOP-USFLJ3H \
LINUX_HOST_HOSTNAME=foobar-ubuntu \
./tools/tuf/test/migration/migration_test.sh
```

To simulate an outage of the TUF during the migration run the above with:
```sh
SIMULATE_NEW_TUF_OUTAGE=1 \
```
