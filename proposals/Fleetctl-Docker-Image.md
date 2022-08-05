# Goal

We need `fleetctl package` functionality to generate all types of packages (PKG, MSI, DEB and RPM) from Linux.

# How

Create a new Docker image `fleetdm/fleetctl` that will contain `fleetctl` and all the dependencies ready to create packages.

Users can then use the image to generate packages
```sh
$ docker run ... fleetdm/fleetctl:latest package --type={pkg|msi|deb|rpm} ...
```

## DEB and RPM

DEB and RPM package generation is already native and no extra dependencies are required (uses https://github.com/goreleaser/nfpm).

## MSI

### Packaging

We will need the same dependencies from `fleetdm/wix:latest` on the new `fleetdm/fleetctl:latest` image.

### Signing (stretch goal)

For `.msi` signing functionality:
- The [relic](https://github.com/sassoftware/relic) tool seems to allow `.msi` signing (in Pure Go).
- Alternatively, the [osslsigncode](https://github.com/mtrojnar/osslsigncode) tool could be embedded on the image.

This is mentioned as a stretch goal because we currently don't have `.msi` signing functionality in `fleetctl package`.

## PKG

### Packaging

To generate a `.pkg` we will need the same dependencies from `fleetdm/bomutils:latest` on the new `fleetdm/fleetctl:latest` image.

### Signing

The [relic](https://github.com/sassoftware/relic) tool seems to allow `.pkg` signing (in Pure Go).

### Notarization

#### Upload

We can implement a Go package that uses the new [Notary API](https://developer.apple.com/documentation/notaryapi) to upload and notarize a `.pkg` (pure Go solution).

#### No Stapling

The Notary API currently does not offer a way to "staple" a package, and the `stapler` tool that allows this is only available on macOS.
It seems stapling is recommended but not a must, see [#116812](https://developer.apple.com/forums/thread/116812).