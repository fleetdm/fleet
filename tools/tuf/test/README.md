# Testing TUF

Scripts in this directory aim to ease the testing of Orbit and the [TUF](https://theupdateframework.io/) system.

> WARNING: All of these scripts are for testing only, they are not safe for production use.

# Setup

1. The script is executed on a macOS host.
2. Fleet server also running on the same macOS host.
3. All VMs (and the macOS host itself) are configured to resolve `host.docker.internal` to the macOS host IP (by modifying their `hosts` file).

> PS: We use `host.docker.internal` because the testing certificate `./tools/osquery/fleet.crt`
> has such hostname (and `localhost`) defined as SANs.

> PPS: Make sure you set the macOSX deployment target to the lowest macOS version you intend to support. See [Troubleshooting](#troubleshooting) for more details.

# Run

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
USE_FLEET_SERVER_CERTIFICATE=1 \
DEBUG=1 \
./tools/tuf/test/main.sh
```

> Separate `*_FLEET_URL` and `*_TUF_URL` variables are defined for each package type to support different setups.

To publish test extensions you can set comma-separated executable paths in the `{MACOS|WINDOWS|LINUX}_TEST_EXTENSIONS` environment variables:
Here's a sample to use the `hello_world` and `hello_mars` test extensions:
```sh
# Build `hello_word` and `hello_mars` test extensions.
./tools/test_extensions/hello_world/build.sh

[...]
MACOS_TEST_EXTENSIONS="./tools/test_extensions/hello_world/macos/hello_world_macos.ext,./tools/test_extensions/hello_world/macos/hello_mars_macos.ext" \
WINDOWS_TEST_EXTENSIONS="./tools/test_extensions/hello_world/windows/hello_world_windows.ext.exe,./tools/test_extensions/hello_world/windows/hello_mars_windows.ext.exe" \
LINUX_TEST_EXTENSIONS="./tools/test_extensions/hello_world/linux/hello_world_linux.ext,./tools/test_extensions/hello_world/linux/hello_mars_linux.ext" \
[...]
./tools/tuf/test/main.sh
```

# Add new updates

To add new updates (osqueryd or orbit), use `push_target.sh`.

E.g. to add a new version of `orbit` for Windows:
```sh
# Compile a new version of Orbit:
GOOS=windows GOARCH=amd64 go build -o orbit-windows.exe ./orbit/cmd/orbit

# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh windows orbit orbit-windows.exe 43
```

If the script was executed on a macOS host, the Orbit binary will be an universal binary. To push updates you can do:

```sh
# Compile a universal binary of Orbit:
CGO_ENABLED=1 \
ORBIT_VERSION=42 \
ORBIT_BINARY_PATH="orbit-macos" \
go run ./orbit/tools/build/build.go

# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh macos orbit orbit-macos 43
```

E.g. to add a new version of `osqueryd` for macOS:
```sh
# Generate osqueryd app bundle.
make osqueryd-app-tar-gz version=5.5.1 out-path=.

# Push the osqueryd target as a new version
./tools/tuf/test/push_target.sh macos-app osqueryd osqueryd.app.tar.gz 5.5.1
```
NOTE: Contributors on macOS with Apple silicon ran into issues running osqueryd downloaded from GitHub. Until this issue is root caused, the workaround is to download osqueryd from [Fleet's TUF](https://tuf.fleetctl.com/).

E.g. to add a new version of `desktop` for macOS:
```sh
# Compile a new version of fleet-desktop
make desktop-app-tar-gz

# Push the desktop target as a new version
./tools/tuf/test/push_target.sh macos desktop desktop.app.tar.gz 43
```

### Troubleshooting

#### Fleet Desktop Startup Issue on macOS

When running Fleet Desktop on an older macOS version than it was compiled on, Orbit may not launch it due to an error:

```
_LSOpenURLsWithCompletionHandler() failed with error -10825
```

Solution: Set the `MACOSX_DEPLOYMENT_TARGET` environment variable to the lowest macOS version you intend to support:

```
export MACOSX_DEPLOYMENT_TARGET=13 # replace '13' with your target macOS version
```
