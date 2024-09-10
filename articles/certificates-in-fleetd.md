# Certificates in fleetd

There are three components in fleetd connecting to the Fleet server using TLS: `orbit`, `Fleet Desktop` and `osqueryd`.
This article aims to describe how TLS CA root certificates are configured in fleetd to connect to a Fleet server securely.

## Default

The default behavior is using the `fleetctl package` command without the `--fleet-certificate` flag.

- By default, `orbit` and `Fleet Desktop` will use the system's CA root store to connect to Fleet.
- `osqueryd` doesn't support using the system's CA root store, it requires passing in a certificate file with the root CA store (via the `--tls_server_certs` flag). The `fleetctl` executable contains an embedded `certs.pem` file generated from https://curl.se/docs/caextract.html [0]. When generating a fleetd package with `fleetctl package` such embedded `certs.pem` file is added to the package [1]. Fleetd configures `osqueryd` to use the `certs.pem` file as CA root store by setting the `--tls_server_certs` argument to such path.

## Using `--fleet-certificate` in `fleetctl package`

When using `--fleet-certificate` in `fleetctl package`, such certificate file is used as a CA root store by `orbit`, `Fleet Desktop` and `osqueryd` (the system's CA store is not used when generating the fleetd package this way).

## Issues with internal and/or intermediates certificates

TLS clients require the CA root and all intermediate certificates that signed the leaf server certificate to be verified.
This means that if the bundled certificate in fleetd [1] doesn't have intermediate certificates that signed the leaf certificate, then the Fleet server will have to be configured to serve the "fullchain".
Here's a list of some scenarios assuming your Fleet server certificate has an intermediate signing certificate:
- ✅ Using fullchain in the Fleet server and root CA only client side.
- ✅ Using fullchain in the Fleet server and root+intermediate bundle client side.
- ✅ Using the leaf certificate in the Fleet server and root+intermediate bundle client side.
- ✅ Using the leaf certificate + intermediate bundle in the Fleet server and root CA only client side.
- ❌ Using the leaf certificate in the Fleet server and root CA only client side. In this scenario the client side (fleetd) doesn't know of the intermediate certificate and thus cannot verify it.

We've seen TLS certificate issues in the following configurations: (for more information see https://github.com/fleetdm/fleet/issues/6085):
- Certificates signed by internal CA/intermediates.
- Certificates issued by Let's Encrypt (that do not serve the fullchain certificate).

When there are certificate issues you will see the following kind of errors in server logs:
```
2024/07/05 15:03:52 http: TLS handshake error from <remote_ip>:<remote_port>: remote error: tls: bad certificate
2024/07/05 15:03:53 http: TLS handshake error from <remote_ip>:<remote_port>: local error: tls: bad record MAC
```
and the following kind of errors on the client side (fleetd):
```
2024-07-05T15:04:52-03:00 DBG get config error="POST /api/fleet/orbit/config: Post \"https://fleet.example.com/api/fleet/orbit/config\": tls: failed to verify certificate: x509: certificate signed by unknown authority"
```
```
W0705 15:16:44.739495 1251102656 init.cpp:760] Error reading config: Request error: certificate verify failed
```

To troubleshoot issues with certificates you can use `fleetctl debug connection` command, e.g.:
```sh
fleetctl debug connection \
  --fleet-certificate ./your-ca-root.pem \
  https://fleet.example.com
```

[0]: We have a Github CI action that runs daily that updates the [certs.pem on the repository](https://github.com/fleetdm/fleet/blob/main/orbit/pkg/packaging/certs.pem) whenever there's a new version of `cacert.pem` in https://curl.se/docs/caextract.html. Such file is embedded into the `fleetctl` executable and used when generating fleetd packages.
[1]: The bundled certificate in fleetd is installed in `/opt/orbit` in macOS/Linux and `C:\Program Files\Orbit` on Windows. By default its name is `certs.pem`, but it will have a different name if the `--fleet-certificate` flag was used when generating the package (`fleetctl package`).


<meta name="articleTitle" value="Certificates in fleetd">
<meta name="authorFullName" value="Lucas Manuel Rodriguez">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-07-09">
<meta name="articleImageUrl" value="../website/assets/images/articles/apple-developer-certificates-on-linux-for-configuration-profile-signing-1600x900@2x.png">
<meta name="description" value="TLS certificates in fleetd">
