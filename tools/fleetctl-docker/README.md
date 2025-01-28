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
make fleetctl-docker
```

#### macOS signing + notarization

To sign and notarize a generated `pkg` you must have:

1. A Developer ID certificate in PEM format
2. An Apple Store Connect API key

> Note: the Developer ID certificate must be in PEM format because this image
> can be run in automated environments where secrets are passed via environment
> variables, and thus they must be in plain text.
>
> To convert a PKCS 12 certificate to PEM, you can run the following command:
>
> ```
> openssl pkcs12 -in /path/to/cert.p12 -out signing-keypair.pem -nodes
> ```

Once you are set, you can build and notarize/staple your package with:

```
docker run -v "$(pwd):/build" fleetdm/fleetctl package --type=pkg   \
  --macos-devid-pem-content="$(cat /path/to/signing-keypair.pem)" \
  --notarize \
  --app-store-connect-api-key-id="A6DX865SKS" \
  --app-store-connect-api-key-issuer="68911d4c-110c-4172-b9f7-b7efa30f9680 " \
  --app-store-connect-api-key-content="$(cat /path/to/AuthKey_A6DX865SKS.p8)"
```

### Publishing

There's a GitHub workflow to build and publish this image to Docker Hub, currently it has to be triggered [manually](https://docs.github.com/en/actions/managing-workflow-runs/manually-running-a-workflow).
