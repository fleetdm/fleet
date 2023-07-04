# TUF Notes

Orbit uses https://theupdateframework.io/ for automatic updates.
This guide has some notes on how Orbit uses such system.

## How fleetctl and orbit use TUF

A TUF client needs trusted signing keys and a URL to fetch new updates.
Both fleetctl (when using the `package` command) and orbit are TUF clients:
- `fleetctl package` uses a TUF server to fetch targets and assemble a installer Orbit package.
- `orbit` uses a TUF server to keep its components up-to-date.

### fleetctl package

To generate installer packages, the `fleetctl package` command needs:
1. Root TUF keys
2. Update URL

By default, fleetctl uses a hardcoded TUF root key and Fleet DM's TUF URL, see [update.go#L32-L33](https://github.com/fleetdm/fleet/blob/6a437aaa358350d88b13905c2dc357c26c065fd5/orbit/pkg/update/update.go#L32-L33).
you can set alternative root TUF keys and update URL via the `--update-roots` and `--update-url` options.

Sample command using alternative root keys and update URL:
```sh
fleetctl package \
    --type=pkg \
    --fleet-url=https://example.com:8080 \
    --enroll-secret=foobar \
    '--update-roots={"signed":{"_type":"root","spec_version":"1.0","version":1,"expires":"2032-10-16T08:09:53-03:00","keys":{"2b757c4827a3bafafff84baee96671d0101d91a71305e897887a7bc23135863d":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"37368304a31a89f84b6c60cf4baeb312036b516cd44584cabd28c748ec7d1acc"}},"4d05ec4fad838337a596ca9488f673828ab4a6f598f960e6bfefa652a94d5e5e":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"ef54804d10c3e76e03289f81897f25495766046badaed98ab74844efb85450e9"}},"603d02b3f0a4b540ad8cfb0650ec2f9818eac55a01faa74fdcb2f7fcee2e99f3":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"6509f680ed6ea7a9196cee411213daede1a94e950ea700c200d6b1de2085e178"}},"81dd8f7c50b98fe1c01c4b77452c459228d064560692d33084cb0b04ea74d5ae":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"2765dcf1630f93fd78a7eb9552ccd2f8a5f6d5697ed74aff8b9dc2ec0e5b476b"}}},"roles":{"root":{"keyids":["603d02b3f0a4b540ad8cfb0650ec2f9818eac55a01faa74fdcb2f7fcee2e99f3"],"threshold":1},"snapshot":{"keyids":["2b757c4827a3bafafff84baee96671d0101d91a71305e897887a7bc23135863d"],"threshold":1},"targets":{"keyids":["4d05ec4fad838337a596ca9488f673828ab4a6f598f960e6bfefa652a94d5e5e"],"threshold":1},"timestamp":{"keyids":["81dd8f7c50b98fe1c01c4b77452c459228d064560692d33084cb0b04ea74d5ae"],"threshold":1}},"consistent_snapshot":false},"signatures":[{"keyid":"603d02b3f0a4b540ad8cfb0650ec2f9818eac55a01faa74fdcb2f7fcee2e99f3","sig":"05292c2c39d5073673a97f2f3b54988e64b9dc8d60eecaf4f2cf575888bd0083b50259df4fa0c33efa8ec528fb4af15ec0c6cd98e4b4b6959b73783bc3a22c06"}]}' \
    --update-url=http://mytuf-server:8081
```

The `fleetctl package` command will trust the provided (or hardcoded) root key and download (+verify) the latest version of the root metadata file from the TUF server. Such file is signed by the root key and specifies the other top-level roles. 

The `tuf-metadata.json` file is placed in the generated installer package (stored in the Orbit root path) and will be used by Orbit at runtime (see below).

### Orbit

Orbit trusts such the packaged `tuf-metadata.json` and uses it as "root of trust" to bootstrap the TUF system. You can also specify an alternative TUF URL via the `--update-url` argument (this is needed in case of domain change of the TUF file server).

#### Edge case when tuf-metadata.json is missing

If the `tuf-metadata.json` file is not in the expected location (e.g. was moved or deleted for some reason), then Orbit will attempt to use the [hard-coded Fleet DM's root key](https://github.com/fleetdm/fleet/blob/6a437aaa358350d88b13905c2dc357c26c065fd5/orbit/pkg/update/update.go#L33) to bootstrap the TUF system. This is handy for systems that use our (Fleet DM) TUF server in case the `tuf-metadata.json` is gone for some reason.