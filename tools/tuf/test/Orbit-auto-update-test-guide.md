# Orbit auto-update test guide

## Setup

To test Orbit we must have a setup for the three OSs where we support Orbit: macOS, Linux and Windows.

This guide assumes:
- A macOS Intel workstation (host), where we'll run most of the commands, TUF server, Orbit and the Fleet server.
- Two VMWare VMs, with Windows 10 and Ubuntu 22.04, where we'll run Orbit.
- The two guest OSs will connect to the host OS via the `host.docker.internal` hostname.
To do this, you can add an entry like `192.168.103.1	host.docker.internal` to the `hosts` file in the VMs
(`/etc/hosts` on Linux and `C:\Windows\System32\drivers\etc\hosts` on Windows).
- The host OS can share packages with the guest OSs (via VMWare's shared folders feature).

## Last release

Head over to https://github.com/fleetdm/fleet/releases and grab the git tag of the last releases for Fleet and Orbit.

At the time of writing:
- Last Orbit release: `orbit-v1.2.0`.
- Last Fleet release: `fleet-v4.21.0`.
- Last osquery version: `5.8.1`. (In the steps below we will use `5.6.0` and `5.7.0` because we need to test auto-updating two times.)

## Run Fleet

```sh
git checkout fleet-v4.21.0

make fleet fleetctl
make db-reset
./build/fleet serve --logging_debug --dev --dev_license

./build/fleetctl setup \
    --email foo@example.com \
    --name foo \
    --password p4ssw0rd.123 \
    --org-name "Fleet Device Management Inc."

export ENROLL_SECRET=K3lOqio9XKw6Cr24qw1XyCRzydwRZeAv
echo "---\napiVersion: v1\nkind: enroll_secret\nspec:\n  secrets:\n  - secret: $ENROLL_SECRET\n" > secrets.yml
./build/fleetctl apply -f secrets.yml
```

## Generate local TUF repository

1. The following commands will generate the TUF repository with the last released version of Orbit and automatically generate the Orbit packages.

```sh
git checkout orbit-v1.2.0
rm -rf test_tuf

SYSTEMS="macos windows linux" \
PKG_FLEET_URL=https://localhost:8080 \
PKG_TUF_URL=http://localhost:8081 \
DEB_FLEET_URL=https://host.docker.internal:8080 \
DEB_TUF_URL=http://host.docker.internal:8081 \
RPM_FLEET_URL=https://host.docker.internal:8080 \
RPM_TUF_URL=http://host.docker.internal:8081 \
MSI_FLEET_URL=https://host.docker.internal:8080 \
MSI_TUF_URL=http://host.docker.internal:8081 \
GENERATE_PKG=1 \
GENERATE_DEB=1 \
GENERATE_RPM=1 \
GENERATE_MSI=1 \
ENROLL_SECRET=$ENROLL_SECRET \
FLEET_DESKTOP=1 \
USE_FLEET_SERVER_CERTIFICATE=1 \
OSQUERY_VERSION=5.6.0 \
./tools/tuf/test/main.sh
```

2. Install Orbit on the host (macOS):
```sh
# (Remove any leftover Orbit from the host.)
orbit/tools/cleanup/cleanup_macos.sh

sudo installer -pkg fleet-osquery.pkg -verbose -target /
```

3. Copy the generated packages into the VMWare shared folders:
```sh
cp fleet-osquery.msi ~/shared-windows
cp fleet-osquery_42.0.0_amd64.deb ~/shared-ubuntu
```

4. Proceed to install Orbit in both VM hosts.
- On the Windows VM:
  - Remove "Fleet osquery" from the installed programs.
  - Double-click the `fleet-osquery.msi` installer to install the new Orbit.
- On Ubuntu:
  ```sh
  # (Remove any leftover Orbit from the host.)
  sudo apt remove fleet-osquery -y
  
  sudo dpkg --install fleet-osquery_42.0.0_amd64.deb
  ```

5. Verify the three hosts have enrolled (by running `./build/fleetctl get hosts` or using the browser).

6. Verify the three Fleet Desktop instances are working, by clicking the "My device" menu item on the three OSs.

## New releases

1. Now let's "release" a new version of the three fleetd components (orbit, Fleet Desktop and osquery) via auto-update by using latest `main`:

```sh
git checkout main
```

### Windows

```sh
# Compile a new version of Orbit for Windows:
GOOS=windows GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit-windows.exe ./orbit/cmd/orbit
# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh windows orbit orbit-windows.exe 43
```

```sh
# Compile a new version of fleet-desktop for Windows:
FLEET_DESKTOP_VERBOSE=1 FLEET_DESKTOP_VERSION=43.0.0 make desktop-windows
# Push the desktop target as a new version
./tools/tuf/test/push_target.sh windows desktop fleet-desktop.exe 43
```

```sh
# Download version from our main TUF:
curl https://tuf.fleetctl.com/targets/osqueryd/windows/5.7.0/osqueryd.exe --output osqueryd.exe

# Push the osqueryd target as a new version
./tools/tuf/test/push_target.sh windows osqueryd osqueryd.exe 5.7.0
```

### Linux

```sh
# Compile a new version of Orbit for Linux:
GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit-linux ./orbit/cmd/orbit
# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh linux orbit orbit-linux 43
```

```sh
# Compile a new version of fleet-desktop for Linux:
FLEET_DESKTOP_VERBOSE=1 FLEET_DESKTOP_VERSION=43.0.0 make desktop-linux
# Push the desktop target as a new version
./tools/tuf/test/push_target.sh linux desktop desktop.tar.gz 43
```

```sh
# Download version from our main TUF:
curl https://tuf.fleetctl.com/targets/osqueryd/linux/5.7.0/osqueryd --output osqueryd-linux

# Push the osqueryd target as a new version
./tools/tuf/test/push_target.sh linux osqueryd osqueryd-linux 5.7.0
```

### macOS

```sh
# Compile a new version of Orbit for macOS:
GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43" -o orbit-darwin ./orbit/cmd/orbit
# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh macos orbit orbit-darwin 43
```

```sh
# Compile a new version of fleet-desktop for macOS:
FLEET_DESKTOP_VERBOSE=1 FLEET_DESKTOP_VERSION=43.0.0 make desktop-app-tar-gz
# Push the desktop target as a new version
./tools/tuf/test/push_target.sh macos desktop desktop.app.tar.gz 43
```

```sh
# Download version from our main TUF.
curl https://tuf.fleetctl.com/targets/osqueryd/macos-app/5.7.0/osqueryd.app.tar.gz --output osqueryd.app.tar.gz

# Push the osqueryd target as a new version
./tools/tuf/test/push_target.sh macos-app osqueryd osqueryd.app.tar.gz 5.7.0
```

2. Wait for ~1m for all hosts to fully auto-update.

3. Run the following live query on all hosts: `SELECT * FROM orbit_info;`. The query should return `version=43`.

4. Verify all hosts now show "Fleet Desktop v43.0.0" on the Fleet Desktop menu.

5. Verify the Fleet Desktop instances are working, by visiting "My device", and hit "Refresh" in the "My device" page.

6. Run the following live query on all hosts: `SELECT * FROM osquery_info;`. The query should return `version=5.7.0`.

## Test Orbit N + 1

We need to test that the `main` (to-be-released) version of Orbit has not broken the auto-update mechanism. So we'll test upgrading to a new hypothetical future version "44".
Run the above tests again on the three OSs but by compiling orbit and Fleet Desktop differently:

- When compiling Orbit, change the following, from:
  ```sh
  -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=43"
  ```
  to
  ```sh
  -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=44"
  ```
- When compiling Fleet Desktop, change the following, from:
  ```sh
  FLEET_DESKTOP_VERSION=43.0.0
  ```
  to
  ```sh
  FLEET_DESKTOP_VERSION=44.0.0
  ```
- When pushing osquery, change `5.7.0` to `5.8.1`.


1. Run the following live query on the hosts: `SELECT * FROM orbit_info;`. The query should now return `version=44`.
2. Run the following live query on the hosts: `SELECT * FROM osquery_info;`. The query should now return `version=5.8.1`.
3. Verify all hosts now show "Fleet Desktop v44.0.0" on the Fleet Desktop menu.

## New Fleet release

1. Kill currently running fleet server instance.

2. Now let's build and "release" latest version of Fleet.
   ```sh
   git checkout main
   make fleet fleetctl
   ./build/fleet prepare db --dev --logging_debug
   ./build/fleet serve --logging_debug --dev --dev_license
   ```

3. Run smoke testing like running a live query on the three hosts to smoke test new Fleet version.

4. Test any new Orbit features added in the release.