# `migration_test.sh`

This script is used to test the migration from one local TUF repository to a new local TUF repository (with new roots).
The "old" TUF will be hosted on port 8081, and the new TUF will be hosted on port 8082.

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

To test TUFs with HTTPS instead of HTTP with two ngrok tunnels that connect to 8081/8082:
```sh
OLD_TUF_URL=https://121e9b4a4dab.ngrok.app \
NEW_TUF_URL=https://12oe8b5b3cc6.ngrok.app \
```

To simulate an outage of the new TUF server during the migration run the above with:
```sh
SIMULATE_NEW_TUF_OUTAGE=1 \
```

To simulate an outage of the new TUF server during the migration and a "need" to patch orbit on the old repository:
```sh
SIMULATE_NEW_TUF_OUTAGE=1 \
ORBIT_PATCH_IN_OLD_TUF=1 \
```
