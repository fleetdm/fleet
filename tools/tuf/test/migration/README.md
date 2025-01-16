# `migration_test.sh`

This script is used to test the migration from one local TUF repository to a new local TUF repository (with new roots).

> Currently supports running on macOS only.

The script is interactive and assumes the user will use a Windows and Ubuntu VM to install fleetd and test the changes on those platforms too.

- `FLEET_URL`: The Fleet server will be hosted on https://localhost:8080, tunneled via ngrok to e.g. https://s123ssfsdgsdf.ngrok.app.
- `OLD_TUF_URL`: The "old" TUF will be hosted on http://localhost:8081, tunneled via ngrok to e.g. https://121e9b4a4dab.ngrok.app.
- `NEW_TUF_URL`: The "new" TUF will be hosted on http://localhost:8082, tunneled via ngrok to e.g. https://12oe8b5b3cc6.ngrok.app.
- `SIMULATE_NEW_TUF_OUTAGE=1`: Simulates an outage of the new TUF server during the migration.
- `ORBIT_PATCH_IN_OLD_TUF=1`: Simulates an outage of the new TUF server during the migration and a "need" to patch orbit on the old repository.
- `WINDOWS_HOST_HOSTNAME`: Hostname of the Windows VM to install fleetd (as reported by osquery/Fleet).
- `LINUX_HOST_HOSTNAME`: Hostname of the Ubuntu VM to install fleetd (as reported by osquery/Fleet).
- `NO_TEAM_ENROLL_SECRET`: Enroll secret of "No team" on your Fleet instance.
```sh
FLEET_URL=https://s123ssfsdgsdf.ngrok.app \
OLD_TUF_URL=https://121e9b4a4dab.ngrok.app \
NEW_TUF_URL=https://12oe8b5b3cc6.ngrok.app \
NO_TEAM_ENROLL_SECRET=... \
WINDOWS_HOST_HOSTNAME=DESKTOP-USFLJ3H \
LINUX_HOST_HOSTNAME=foobar-ubuntu \
SIMULATE_NEW_TUF_OUTAGE=1 \
ORBIT_PATCH_IN_OLD_TUF=1 \
./tools/tuf/test/migration/migration_test.sh
```