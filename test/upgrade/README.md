# Upgrade Tests

This tool can be used to test DB upgrades between two Fleet versions.

To run the tests, you need to specify the "from" and "to" versions, for example:
```sh
FLEET_VERSION_A=v4.16.0 FLEET_VERSION_B=v4.18.0 go test ./test/upgrade
```

Ensure that Docker is installed with Compose V2.
To check if you have the correct version, run the following command
```sh
docker compose version
Docker Compose version v2.6.0
```
