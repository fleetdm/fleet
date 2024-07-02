# fleetd certificates

There are three components in fleetd connecting to the Fleet server (TLS): `orbit`, `Fleet Desktop` and `osqueryd`.
This document aims to describe how TLS CA root certificates are configured in fleetd to connect to a Fleet server securely.

## Default

The default behavior is using the `fleetctl package` command without the `--fleet-certificate` flag.

- By default, `orbit` and `Fleet Desktop` will use the system's CA root store to connect to Fleet.
- `osqueryd` doesn't support using the system's CA root store, it requires passing in a certificate file with the root CA store (via the `--tls_server_certs` flag). The `fleetctl` executable contains an embedded `certs.pem` file generated [0] from https://curl.se/docs/caextract.html. When generating a fleetd package with `fleetctl package` such embedded `certs.pem` file is added to the package. When installing the fleetd package, such file is installed in `/opt/bin` on Linux/macOS and `C:\Program Files\Orbit` on Windows. Fleetd configures `osqueryd` to use the `certs.pem` file as CA root store by setting the `--tls_server_certs` argument to such path.

## Using `--fleet-certificate` in `fleetctl package`

When using `--fleet-certificate` in `fleetctl package`, such certificate file is used as a CA root store by `orbit`, `Fleet Desktop` and `osqueryd` (the system's CA store is not used when generating the fleetd package this way).

## Issues with internal/intermediates certificates

Fleetd requires the CA root and all intermediate certificates that signed the server certificate to be present in its bundled certificate and may require the fullchain (root CA + intermediates) be configured in the server. This is usually the case with certificates signed by internal CA/intermediates and we've also seen some issues with certificates issued by Let's Encrypt (see https://github.com/fleetdm/fleet/issues/6085).

To troubleshoot issues with certificates you can use `fleetctl debug connection` command, e.g.:
```sh
fleetctl debug connection \
  --fleet-certificate ./your-ca-root.pem \
  https://fleet.example.com
```

[0] We have a Github CI action that runs daily that updates the [certs.pem on the repository](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/packaging/certs.pem) whenever there's a new version of `cacert.pem` in https://curl.se/docs/caextract.html. Such file is embedded into the `fleetctl` executable and used when generating fleetd packages.