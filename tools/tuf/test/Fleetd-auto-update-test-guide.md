# Fleetd auto-update N+1 test

The following guide describes how to test an N+1 upgrade to fleetd components.
We need to test that the `main` (to-be-released) version of fleetd has not broken the auto-update mechanism.

> This guide only supports running from a macOS workstation.

## Setup

Follow the setup in the [README.md](./README.md).

## Build and push fleetd N+1

### orbit

Build:
```sh
GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit-darwin ./orbit/cmd/orbit
GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit-linux ./orbit/cmd/orbit
GOOS=windows GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit.exe ./orbit/cmd/orbit
```
Push:
```sh
./tools/tuf/test/push_target.sh macos orbit orbit-darwin 43
./tools/tuf/test/push_target.sh linux orbit orbit-linux 43
./tools/tuf/test/push_target.sh windows orbit orbit.exe 43
```

### desktop

Build:
```sh
FLEET_DESKTOP_VERSION=43 make desktop-app-tar-gz
FLEET_DESKTOP_VERSION=43 make desktop-windows
FLEET_DESKTOP_VERSION=43 make desktop-linux
```
```sh
./tools/tuf/test/push_target.sh macos desktop desktop.app.tar.gz 43
./tools/tuf/test/push_target.sh linux desktop desktop.tar.gz 43
./tools/tuf/test/push_target.sh windows desktop fleet-desktop.exe 43
```

### osqueryd

Assuming we are upgrading to 5.11.0 (you can also downgrade to a lower version to test the auto-update mechanism)

Download:
```sh
# macOS
make osqueryd-app-tar-gz version=5.11.0 out-path=.

# osqueryd
curl -L https://github.com/osquery/osquery/releases/download/5.11.0/osquery_5.11.0-1.linux_amd64.deb --output osquery.deb
ar x osquery.deb
tar xf data.tar.gz
chmod +x ./opt/osquery/bin/osqueryd
cp ./opt/osquery/bin/osqueryd osqueryd

# Windows
curl -L https://github.com/osquery/osquery/releases/download/5.11.0/osquery-5.11.0.msi --output osquery-5.11.0.msi
# Run the following on a Windows device:
msiexec /a osquery-${{ env.OSQUERY_VERSION }}.msi /qb TARGETDIR=C:\temp
# Copy C:\temp\osquery\osqueryd\osqueryd.exe from the Windows device into the macOS workstation.
```
Release:
```sh
./tools/tuf/test/push_target.sh macos-app osqueryd osqueryd.app.tar.gz 5.11.0
./tools/tuf/test/push_target.sh linux osqueryd ./osqueryd 5.11.0
./tools/tuf/test/push_target.sh windows osqueryd ./osqueryd.exe 5.11.0
```

## Verify auto-update

1. Run the following live query on the hosts: `SELECT * FROM orbit_info;`. The query should now return `version=43`.
2. Run the following live query on the hosts: `SELECT * FROM osquery_info;`. The query should now return `version=5.11.0`.
3. Verify all hosts now show "Fleet Desktop v43.0.0" on the Fleet Desktop menu.