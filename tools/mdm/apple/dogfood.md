# Guide for Infrastructure Team

## Memory requirements

Fleet and MySQL servers will need +500 MB extra of memory.

## MySQL

MySQL must be run with `--max_allowed_packet=536870912` // 512 MB

## Configuration

Apple MDM is enabled with the following configuration:

```sh
FLEET_MDM_APPLE_ENABLE=1
```

Additional configuration is generated using `fleetctl`. These credentials are highly sensitive and should be stored securely (e.g. on AWS secretsmanager) and provided to Fleet via environment variables.
Also, ensure that `server_settings.server_url` is set to the public URL of the Fleet deployment. This should already be the case. 

### SCEP

Generate SCEP CA certificate and key:
```sh
fleetctl apple-mdm setup scep \
    --validity-years=5 \
    --cn "FleetDM" \
    --organization "Fleet Device Management Inc." \
    --organizational-unit "Fleet Device Management Inc." \
    --country US
```
The content of such generated files must be stored securely and then fed to Fleet via the following environment variables:
```sh
FLEET_MDM_APPLE_SCEP_CA_CERT_PEM=<contents of SCEP CA certificate>
FLEET_MDM_APPLE_SCEP_CA_KEY_PEM=<contents of SCEP CA certificate key>
```

We also need to generate a random passphrase and store it somewhere (it's less sensitive than the other credentials defined herein, but for consistency it could be stored securely).
```
FLEET_MDM_APPLE_SCEP_CHALLENGE=<some random text>
```

For example, the challenge can be generated using `openssl`
```sh
openssl rand -base64 24
```

### APN

Zach Wasserman will provide the Apple Push Notification service (APNs) certificate and key. The contents must be stored securely and be provided to Fleet via the following environment variables:
```sh
FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM=<contents of APNs certificate>
FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM=<contents of APNs certificate key>
```

### DEP

Follow the instructions in [DEP setup](https://github.com/fleetdm/fleet/blob/apple-mdm/tools/mdm/apple/demo.md#4-dep-setup).
The output is a `fleet-mdm-apple-dep.token` file which contents must be stored securely and then provided to Fleet via an environment variable:
```sh
FLEET_MDM_APPLE_DEP_TOKEN=<contents of DEP token>
```
