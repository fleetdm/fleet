# Testing TUF

Scripts in this directory aim to ease the testing of Orbit and the [TUF](https://theupdateframework.io/) system.

> WARNING: All of these scripts are for testing only, they are not safe for production use.

# Setup

1. The script is executed on a macOS host.
2. Fleet server also running on the same macOS host.
3. All VMs (and the macOS host itself) are configured to resolve `host.docker.internal` to the macOS host IP (by modifying their `hosts` file).
4. The hosts are running on the same GOARCH as the macOS host. If not, you can set the `GOARCH` environment variable to compile for the desired architecture. For example: `GOARCH=amd64`

> PS: We use `host.docker.internal` because the testing certificate `./tools/osquery/fleet.crt`
> has such hostname (and `localhost`) defined as SANs.

> PPS: Make sure you set the macOSX deployment target to the lowest macOS version you intend to support. See [Troubleshooting](#troubleshooting) for more details.

# Run

The `main.sh` creates and runs the TUF repository and optionally generate the installers (GENERATE_PKGS):
```sh
SYSTEMS="macos windows linux linux-arm64 windows-arm64" \
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
GENERATE_DEB_ARM64=1 \
GENERATE_RPM=1 \
GENERATE_RPM_ARM64=1 \
GENERATE_MSI=1 \
GENERATE_MSI_ARM64=1 \
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
WINDOWS_ARM64_TEST_EXTENSIONS="./tools/test_extensions/hello_world/windows-arm64/hello_world_windows_arm64.ext.exe,./tools/test_extensions/hello_world/windows-arm64/hello_mars_windows_arm64.ext.exe" \
LINUX_TEST_EXTENSIONS="./tools/test_extensions/hello_world/linux/hello_world_linux.ext,./tools/test_extensions/hello_world/linux/hello_mars_linux.ext" \
LINUX_ARM64_TEST_EXTENSIONS="./tools/test_extensions/hello_world/linux-arm64/hello_world_linux_arm64.ext,./tools/test_extensions/hello_world/linux-arm64/hello_mars_linux_arm64.ext" \
[...]
./tools/tuf/test/main.sh
```

To build for a specific architecture, you can pass the `GOARCH` environment variable:
``` shell
[...]
# defaults to amd64
GOARCH=arm64 \
[...]
./tools/tuf/test/main.sh
```

# Test fleetd with expired signatures on a TUF repository

To generate a TUF repository with shorter expiration time for roles you can set the following environment variables:
```shell
[...]
KEY_EXPIRATION_DURATION=5m \
TARGETS_EXPIRATION_DURATION=5m \
SNAPSHOT_EXPIRATION_DURATION=5m \
TIMESTAMP_EXPIRATION_DURATION=5m \
[...]
./tools/tuf/test/main.sh
```

> NOTE: The duration has to be enough time to generate the packages (otherwise the `fleetctl package` command will fail).

> `KEY_EXPIRATION_DURATION` is used to set the expiration of the `root.json` signature.

# Add new updates

To add new updates (osqueryd or orbit), use `push_target.sh`.

E.g. to add a new version of `orbit` for Windows:
```sh
source ./tools/tuf/test/load_orbit_version_vars.sh

# Compile a new version of Orbit:
GOOS=windows GOARCH=amd64 go build \
    -o orbit-windows.exe \
    -ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION \
    -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" \
    ./orbit/cmd/orbit

# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh windows orbit orbit-windows.exe $ORBIT_VERSION
```

If the script was executed on a macOS host, the Orbit binary will be a universal binary. To push updates you can do:

```sh
source ./tools/tuf/test/load_orbit_version_vars.sh

# Compile a universal binary of Orbit:
CGO_ENABLED=1 \
ORBIT_VERSION=$ORBIT_VERSION \
ORBIT_COMMIT=$ORBIT_COMMIT \
ORBIT_BINARY_PATH="orbit-macos" \
go run ./orbit/tools/build/build.go

# Push the compiled Orbit as a new version
./tools/tuf/test/push_target.sh macos orbit orbit-macos $ORBIT_VERSION
```

E.g. to add a new version of `osqueryd` for macOS:
```sh
# Generate osqueryd app bundle.
make osqueryd-app-tar-gz version=5.5.1 out-path=.

# Push the osqueryd target as a new version
./tools/tuf/test/push_target.sh macos-app osqueryd osqueryd.app.tar.gz 5.5.1
```
NOTE: Contributors on macOS with Apple silicon ran into issues running osqueryd downloaded from GitHub. Until this issue is root caused, the workaround is to download osqueryd from [Fleet's TUF](https://updates.fleetdm.com/).

E.g. to add a new version of `desktop` for macOS:
```sh
source ./tools/tuf/test/load_orbit_version_vars.sh

# Compile a new version of fleet-desktop
FLEET_DESKTOP_VERSION=$ORBIT_VERSION make desktop-app-tar-gz

# Push the desktop target as a new version
./tools/tuf/test/push_target.sh macos desktop desktop.app.tar.gz $ORBIT_VERSION
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

#### Issue generating linux-arm64 packages when running Docker Desktop on macOS using Apple Silicon

When running Docker Desktop on macOS using Apple Silicon, enrollment packages for ARM Linux may fail to generate and you may see a warning similar to:

```
WARNING: The requested image's platform (linux/amd64) does not match the detected host platform (linux/arm64/v8) and no specific platform was requested
[...]
/usr/local/go/pkg/tool/linux_amd64/compile: signal: illegal instruction
make: *** [desktop-linux] Error 1
```

Solution: In Docker Desktop go to Settings >> General >> Virtual Machine Options and choose the "Docker VMM (BETA)" option. Restart Docker Desktop.

#### Running without ssl

If you decide that you want to run your local fleet server with the `--server_tls=false` flag you will need to modify a few ENV variables when running the `./tools/tuf/test/main.sh` file.

```
+ INSECURE=1 \
- USE_FLEET_SERVER_CERTIFICATE=1 \

+ PKG_FLEET_URL=http://localhost:8080 \
- PKG_FLEET_URL=https://localhost:8080 \

+ DEB_FLEET_URL=http://host.docker.internal:8080 \
- DEB_FLEET_URL=https://host.docker.internal:8080 \

+ RPM_FLEET_URL=http://host.docker.internal:8080 \
- RPM_FLEET_URL=https://host.docker.internal:8080 \

+ MSI_FLEET_URL=http://host.docker.internal:8080 \
- MSI_FLEET_URL=https://host.docker.internal:8080 \
```

These flags change the way `tools/tuf/test/gen_pkgs.sh` builds the binaries to properly support a local server not running ssl.