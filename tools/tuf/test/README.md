# Testing TUF

Scripts in this directory aim to ease the testing of Orbit and the [TUF](https://theupdateframework.io/) system.

> WARNING: All of these scripts are for testing only, they are not safe for production use.

# Setup and Run

The `main.sh` creates and runs the TUF repository and optionally generate the installers (GENERATE_PKGS):
```sh
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
ENROLL_SECRET=6/EzU/+jPkxfTamWnRv1+IJsO4T9Etju \
FLEET_DESKTOP=1 \
FLEET_CERTIFICATE=1 \
./tools/tuf/test/main.sh
```

Separate `*_FLEET_URL` and `*_TUF_URL` variables are needed for each package to support different setups.
E.g. The values shown above assume:
1. The script is executed on a macOS host.
2. Fleet server also running on the same macOS host.
3. All VMs (and the macOS host itself) are configured to resolve `host.docker.internal` to the macOS host IP (by modifying their `hosts` file).

> PS: We use `host.docker.internal` because the testing certificate `./tools/osquery/fleet.crt`
> has such hostname (and `localhost`) defined as SANs.

# Add new updates

To add new updates (osqueryd or orbit), use `push_target.sh`.

E.g. to add a new version of `orbit` for Windows:
```sh
# Compile a new version of Orbit:
GOOS=windows GOARCH=amd64 go build -o orbit-windows.exe ./orbit/cmd/orbit

# Push the compiled Orbit as a new version:
./tools/tuf/test/push_target.sh windows orbit orbit-windows.exe 43
```

E.g. to add a new version of `osqueryd` for macOS:
```sh
# Download some version from our TUF server:
curl --output osqueryd https://tuf.fleetctl.com/targets/osqueryd/macos/5.0.1/osqueryd

# Push the osqueryd target as a new version:
./tools/tuf/test/push_target.sh macos osqueryd osqueryd 43
```