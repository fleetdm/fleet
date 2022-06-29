## fleetdm/fleetctl

This docker image allows to run `fleetctl` in a Linux environment that has all
the necessary dependencies to package `msi`, `pkg`, `deb` and `rpm` packages.

### Usage

```
docker run fleetdm/fleetctl command [flags]
```

Build artifacts are generated at `/build`. To get a package using this image:

```
docker run -v "$(pwd):/build" fleetdm/fleetctl package --type=msi
```

### Building

This image needs to be built from the root of the repo in order for the build
context to have access to the `fleetctl` binary. To build the image, run:

```
make xp-fleetctl
docker build -t fleetdm/fleetctl --platform=linux/amd64 -f tools/fleetctl-docker/Dockerfile .
```
