# test-certs

This directory contains a fake certificate chain to test TLS functionality in `fleet`, `fleetctl` and `fleetd`.

> The certificates were generated using the following guide: [OpenSSL create certificate chain](https://www.golinuxcloud.com/openssl-create-certificate-chain-linux/#Step_6_Generate_and_sign_server_certificate_using_Intermediate_CA)

## Directories

### root-ca directory

Contains a self-signed certificate considered as the "root CA" certificate.

### intermediate-ca directory

Contains a certificate signed by the "root CA" and considered as the "intermediate CA" certificate.
Additionaly contains a `intermediate-and-root.cert.pem` which contains `intermediate.cert.pem` + `root-ca.cert.pem`.

### server

Contains a server certificate signed by the "intermediate CA" certificate.

Contains certificates that can be used by a Fleet server:
- `server.key.pem`: TLS server private key.
- `leaf.cert.pem`: TLS server certificate alone.
- `leaf-and-intermediate.cert.pem`: Contains `leaf.cert.pem` + `intermediate.cert.pem`.
- `fullchain.cert.pem`: Contains `leaf.cert.pem` + `intermediate-ca.cert.pem` + `root-ca.crt.pem`.

## Usage

Run the Fleet server with the leaf certificate only:
```sh
fleet serve --dev --dev_license \
    --server_cert ./tools/test-certs/server/leaf.cert.pem \
    --server_key ./tools/test-certs/server/server.key.pem \
    --logging_debug
```

You will see that `fleetctl debug connection` will fail if only pinning the `root-ca.cert.pem` (because TLS client doesn't know about the intermediate certificate):
```sh
fleetctl debug connection \
    --fleet-certificate ./tools/test-certs/root-ca/root-ca.cert.pem \
    https://localhost:8080
Debugging connection to localhost; Configuration context: none - using provided address; Root CA: ./tools/test-certs/root-ca/root-ca.cert.pem; TLS: secure.
Success: can resolve host localhost.
Success: can dial server at localhost:8080.
Error: Fail: certificate: dial for validate: verify certificate: x509: certificate signed by unknown authority
```

And `fleetctl debug connection` will succeed if pinning with `intermediate-and-root.cert.pem`:
```sh
fleetctl debug connection --fleet-certificate ./tools/test-certs/intermediate-ca/intermediate-and-root.cert.pem https://localhost:8080
Debugging connection to localhost; Configuration context: none - using provided address; Root CA: ./tools/test-certs/intermediate-ca/intermediate-and-root.cert.pem; TLS: secure.
Success: can resolve host localhost.
Success: can dial server at localhost:8080.
Success: TLS certificate seems valid.
Success: agent API endpoints are available.
```
