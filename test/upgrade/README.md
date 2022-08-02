# Upgrade Tests

The tests located in `test/upgrade` are intended to test fleet upgrades with online migrations as proposed in [#6376](https://github.com/fleetdm/fleet/pull/6376).
To run the tests, you need to specify the from and to versions. For example

```
$ FLEET_VERSION_A=v4.16.0 FLEET_VERSION_B=v4.18.0 go test ./test/upgrade
```

Ensure that Docker is installed with Compose V2.
To check if you have the correct version, run the following command

```
$ docker compose version
Docker Compose version v2.6.0
```
