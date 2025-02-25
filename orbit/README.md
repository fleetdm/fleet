<img width="200" alt="Fleet logo, landscape, dark text, transparent background" src="https://user-images.githubusercontent.com/618009/103300491-9197e280-49c4-11eb-8677-6b41027be800.png">

Orbit is a lightweight osquery installer and autoupdater. With Orbit, it's easy to deploy osquery, manage configurations, and keep things up-to-date. Orbit eases the deployment of osquery connected with a [Fleet server](https://github.com/fleetdm/fleet), and is a (near) drop-in replacement for osquery in a variety of deployment scenarios.

Orbit is the recommended agent for Fleet. But Orbit can be used with or without Fleet, and Fleet can be used with or without Orbit.

# Documentation

- [Releasing Orbit](docs/Releasing-Orbit.md)

## How to build from source

To build orbit we use [goreleaser](https://goreleaser.com/).

For reference, here are the build configuration files:
- [Goreleaser github workflow](../.github/workflows/goreleaser-orbit.yaml)
- Goreleaser configuration file for each platform:
    - [goreleaser-linux.yml](./goreleaser-linux.yml)
    - [goreleaser-linux-arm64.yml](./goreleaser-linux-arm64.yml)
    - [goreleaser-macos.yml](./goreleaser-macos.yml)
    - [goreleaser-windows.yml](./goreleaser-windows.yml)

Following are the commands to build in case you can't use goreleaser.

> IMPORTANT: We recommend you build orbit natively and not cross compile to avoid any build or runtime errors.

### macOS
```sh
CGO_ENABLED=1 \
CODESIGN_IDENTITY=$CODESIGN_IDENTITY \
ORBIT_VERSION=$VERSION \
ORBIT_BINARY_PATH=./orbit-macos \
go run ./orbit/tools/build/build.go
```

### Windows
```sh
CGO_ENABLED=0 \
GOOS=windows \
GOARCH=amd64 \
go build \
-trimpath \
-ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$VERSION \
-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$COMMIT \
-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Date=$DATE" \
-o ./orbit.exe ./orbit/cmd/orbit
```

### Linux
```sorbit/README.mdh
CGO_ENABLED=1 \
GOOS=linux \
GOARCH=amd64 \
go build \
-trimpath \
-ldflags="-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$VERSION \
-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$COMMIT \
-X github.com/fleetdm/fleet/v4/orbit/pkg/build.Date=$DATE" \
-o ./orbit-linux ./orbit/cmd/orbit
```

## Bugs

To report a bug or request a feature, [click here](https://github.com/fleetdm/fleet/issues).

## Orbit Development

### Run Orbit From Source

To execute orbit from source directly, run the following command:

```sh
go run github.com/fleetdm/fleet/v4/orbit/cmd/orbit \
    --dev-mode \
    --disable-updates \
    --root-dir /tmp/orbit \
    --fleet-url https://localhost:8080 \
    --insecure \
    --enroll-secret Pz3zC0NMDdZfb3FtqiLgwoexItojrYh/ \
    -- --verbose
```

Or, using a `flagfile.txt` for osqueryd:
```sh 
go run github.com/fleetdm/fleet/v4/orbit/cmd/orbit \
    --dev-mode \
    --disable-updates \
    --root-dir /tmp/orbit \
    -- --flagfile=flagfile.txt --verbose
```

### Generate Installer Packages from Orbit Source

The `fleetctl package` command generates installers by fetching the targets/executables from a [TUF](https://theupdateframework.io/) repository.
To generate an installer that contains an Orbit built from source you need to setup a local TUF repository.
The following document explains how you can generate a TUF repository, and installers that use it [tools/tuf/test](../tools/tuf/test/README.md).

## FAQs

### How does Orbit compare with Kolide Launcher?

Orbit is inspired by the success of [Kolide Launcher](https://github.com/kolide/launcher), and approaches a similar problem domain with new strategies informed by the challenges encountered in real world deployments. Orbit does not share any code with Launcher.

- Both Orbit and Launcher use [The Update Framework](https://theupdateframework.com/) specification for managing updates. Orbit utilizes the official [go-tuf](https://github.com/theupdateframework/go-tuf) library, while Launcher has it's own implementation of the specification.
- Orbit can be deployed as a (near) drop-in replacement for osquery, supporting full customization of the osquery flags. Launcher heavily manages the osquery flags making deployment outside of Fleet or Kolide's SaaS difficult.
- Orbit prefers the battle-tested plugins of osquery. Orbit uses the built-in logging, configuration, and live query plugins, while Launcher uses custom implementations.
- Orbit prefers the built-in osquery remote APIs. Launcher utilizes a custom gRPC API that has led to issues with character encoding, load balancers/proxies, and request size limits.
- Orbit encourages use of the osquery performance Watchdog, while Launcher disables the Watchdog.

Additionally, Orbit aims to tackle problems out of scope for Launcher:

- Configure updates via release channels, providing more granular control over agent versioning.
- Manage osquery startup flags from a remote (Fleet) server.
- Support for deploying and updating osquery extensions (🔜).
- Manage osquery versions from a remote (Fleet) server (🔜).

### Is Orbit Free?

Yes! Orbit is licensed under an MIT license and all uses are encouraged.

### How does orbit update osquery? And how do the stable and edge channels get triggered to update osquery on a self hosted Fleet instance?

Orbit uses a configurable update server. We expect that many folks will just use the update server we manage (similar to what Kolide does with Launcher's update server). We are also offering [tooling for self-managing an update server](https://fleetdm.com/docs/deploying/fleetctl-agent-updates) as part of Fleet Premium (the subscription offering).

## Community

### Chat

Please join us in the #fleet channel on [osquery Slack](https://fleetdm.com/slack).

<a href="https://fleetdm.com"><img alt="Banner featuring a futuristic cloud city with the Fleet logo" src="https://user-images.githubusercontent.com/618009/98254443-eaf21100-1f41-11eb-9e2c-63a0545601f3.jpg"/></a>
