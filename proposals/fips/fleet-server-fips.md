# Planning for a FIPS Compliant Fleet Server for Linux

This document outlines the steps involved in building a version of the Fleet server that meets the requirements of the Federal Information Processing Standards (FIPS) 140-2 security standard.

## Summary of cryptographic operations in Fleet

To make the Fleet server FIPS 140-2 compliant, all cryptographic operations must use a FIPS 140-2 certified cryptographic library.

Fleet uses cryptographic operations in the following features:
- Fleet uses TLS as a client for the following features:
    - Fleet can conect to MySQL and Redis using TLS.
    - Vulnerability processing uses TLS client to retrieve data from different sources.
    - Automations: Webhooks, JIRA and Zendesk integrations make use of a TLS client.
    - Single Sign On uses TLS to retrieve IdP metadata (from the configured MetadataURL).
    - MDM functionality connects via TLS to Apple servers (DEP enrollment configuration).
    - Fleet uses TLS to stream logs to many data stream systems (Kafka, Lambda, Kinesis, Firehose, etc.).
- Fleet can act as a TLS server (can terminate TLS).
- User authentication: Fleet uses password hashing to store and authenticate user credentials. 
- The Single Sign On (SSO) feature in Fleet performs cryptographic verification of `SAMLResponse`s signatures. Fleet uses [goxmldsig](https://github.com/russellhaering/goxmldsig) which uses golang's stdlib crypto (`crypto/x509`, `crypto/rsa`, `crypto/sha256`, `crypto/tls` which are all compliant primitives)
- Fleet uses `crypto/rand` to generate secrets for authentication ("session tokens" and "node keys" which are used for authenticating users and devices respectively).
- MDM functionality (for Windows and macOS) makes use of cryptographic operations (e.g. Fleet acts as a SCEP CA server for authenticating devices)
- Fleet license check code uses `JWT` which uses `ECDSA` for public signatures (ECDSA is a FIPS-compliant primitive).

All these operations (except password hashing, see [Password hashing](#password-hashing) below) make use of Go's `crypto` standard library.

## Building Go with BoringSSL as cryptographic backend

As of today, the recommended way to make your Go application be FIPS 140-2 compliant is to use the BoringSSL crypto backend instead of the standard library `crypto` packages.

Since we use Go 1.19, we only need to build fleet with `CGO_ENABLED=1` (already the case because of our sqlite3 dependency) and with `GOEXPERIMENT=boringcrypto`. This will automatically make Fleet use the BoringSSL cryptographic primitives instead of the stdlib `crypto` implementation (without requiring any code changes).

See the [POC]https://github.com/fleetdm/fleet/compare/13288-poc-fleet-fips?expand=1()).

> Source: https://kupczynski.info/posts/fips-golang/

## Password hashing

For password hashing Fleet currently uses the [bcrypt](https://en.wikipedia.org/wiki/Bcrypt) function.
To be FIPS 140-2 compliant, Fleet will have to use [PBKDF2](https://en.wikipedia.org/wiki/PBKDF2] instead (with SHA-256 as hashing primitive).

> Source: https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-132.pdf

## Summary of tasks

Assumptions around the first customer using the FIPS build:
- All MDM funtionality will be disabled/not-used.
- The Fleet server won't be doing TLS termination. Thus we don't need to verify/test such feature.
- This build will be used as part of a new deployment (not migrating an existing one). This is important because we have to change the password hashing algorithm thus to reduce complexity on the first iteration we don't need to worry about migrating from old to new password hashing.
- The `fleetctl` command is outside the scope.

Tasks:

- Double check if any cryptographic operations in Fleet were missed in this analysis. // 1 pt.
- Add a new target `fleet-fips` to the `Makefile` to build Fleet in FIPS mode. (Would set `GOEXPERIMENT=boringcrypto CGO_ENABLED=1`.) Smoke test the Fleet server when built this way. (See the [POC]https://github.com/fleetdm/fleet/compare/13288-poc-fleet-fips?expand=1()) // 1 pt.
- Make changes in goreleaser yamls to create a new docker image fleetdm/fleet-fips // 2 pt.
- Perform full QA of the Feet docker image FIPS build (by full QA we mean: we need to test ALL Fleet features). This should be performed by a QA engineer. // 5 pt.
- Perform any code changes to fleetd (and add documentation for vanilla osquery) to be able to connect (via TLS) to a FIPS-compliant Fleet. We will know if this is necessary from the previous task (QA). // 2 pt.
- Add tests to make sure the Fleet server TLS client is using the FIPS approved ciphers when connecting to TLS servers (SSO, users, vuln processing, webhooks). E.g. by using TLS servers that will fail the connection if the FIPS ciphers are not used. // 3 pt.
- Replace bcrypt with PBKDF2 for password hashing (when compiling in FIPS mode) // 5 pt. (assumming we don't need to migrate from bcrypt)
- Terminate the Fleet sever if MDM or TLS server is enabled when running the FIPS build (security measure). // 1 pt.
- Check if `crypto/subtle` (used by the `/metrics` endpoint for HTTP basic auth) is implemented by BoringSSL. // 1 pt.
- Loadtest the Fleet server with expected number of devices. // 5 pt. (depends on number of hosts)
- Deploy the Fleet FIPS docker image and dependencies to a FIPS enabled AWS endpoint // 5 pt.
