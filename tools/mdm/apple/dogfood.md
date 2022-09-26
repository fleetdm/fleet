# Guide for Infrastructure Team

## Memory requirements

Fleet and MySQL servers will need +500 MB extra of memory.

## MySQL

MySQL must be run with `--max_allowed_packet=536870912` // 512 MB

## New Credentials and Configuration

The following are new credentials that are generated via `fleetctl`, must be stored securely (e.g. on AWS secretmanager), and are fed to Fleet via environment variables.

### SCEP

Generate SCEP CA certificate and key:
```sh
fleetctl apple-mdm setup scep --validity-years=5 --cn "FleetDM" --organization "Fleet Device Management Inc." --organizational-unit "Fleet Device Management Inc." --country US
Successfully generated SCEP CA: fleet-mdm-apple-scep.crt, fleet-mdm-apple-scep.key.
Set FLEET_MDM_APPLE_SCEP_CA_CERT_PEM=$(cat fleet-mdm-apple-scep.crt) FLEET_MDM_APPLE_SCEP_CA_KEY_PEM=$(cat fleet-mdm-apple-scep.key) when running Fleet.
```
The content of such generated files must be stored securely and then fed to Fleet via the following environment variables:
```sh
FLEET_MDM_APPLE_SCEP_CA_CERT_PEM=$(cat fleet-mdm-apple-scep.crt)
FLEET_MDM_APPLE_SCEP_CA_KEY_PEM=$(cat fleet-mdm-apple-scep.key)
```

We also need to generate a random passphrase and store it somewhere (it's less sensitive than the other credentials defined herein, but for consistency it could be stored securely).
Such passphrase is passed also as enviroment variable to Fleet:
```
FLEET_MDM_APPLE_SCEP_CHALLENGE=<some_secret_passphrase>
```

### APNs

Zach Wasserman will provide `from.zach.push.pem` and `from.zach.push.key`, content of such files must be stored securely and then fed to Fleet via the following environment variables:
```sh
FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM=$(cat from.zach.push.pem)
FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM=$(cat from.zach.push.key)
```

### DEP

Run (via `fleetctl`) the "DEP Setup" defined in [demo.md](https://github.com/fleetdm/fleet/blob/apple-mdm/tools/mdm/apple/demo.md#4-dep-setup).
Output is a `fleet-mdm-apple-dep.token` file which contents must be stored securely and then fed to Fleet via environment variable:
```sh
FLEET_MDM_APPLE_DEP_TOKEN=$(cat fleet-mdm-apple-dep.token)
```

## Run Fleet

Apart from the above configuration, the following must be set to run Fleet with Apple MDM:
```
FLEET_MDM_APPLE_ENABLE=1
FLEET_MDM_APPLE_SERVER_ADDRESS=dogfood.fleetdm.com // i.e. public server address of the Fleet deployment
```