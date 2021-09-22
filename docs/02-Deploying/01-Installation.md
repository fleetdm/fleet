# Installation

- [Installing the Fleet binary](#installing-the-fleet-binary)
  - [Docker container](#docker-container)
  - [Raw binaries](#raw-binaries)
- [TLS configuration](#tls-configuration)
  - [TLS certificate considerations](#tls-certificate-considerations)
- [Infrastructure dependencies](#infrastructure-dependencies)
  - [MySQL](#mysql)
  - [Redis](#redis)

The Fleet application is distributed as a single static binary. This binary serves:

- The Fleet web interface
- The Fleet application API endpoints
- The osquery TLS server API endpoints

All of these are served via a built-in HTTP server, so there is no need for complex web server configurations. Once you've installed the `fleet` binary and it's infrastructure dependencies as illustrated below, refer to the [Configuration](./2-Configuration.md) documentation for information on how to use and configure the Fleet application.

## Installing the Fleet binary

There are multiple options available for installing the Fleet binary.

### Docker container

Pull the latest Fleet docker image:

```
docker pull fleetdm/fleet
```

For more information on using Fleet, refer to the [Configuration](./2-Configuration.md) documentation.

### Raw binaries

Download the latest raw Fleet binaries with `curl` or from the ["Releases" page on GitHub](https://github.com/fleetdm/fleet/releases).

Unzip the binaries for your platform:

```
# For a Darwin compatible binary
unzip fleet.zip 'darwin/*' -d fleet
./fleet/darwin/fleet_darwin_amd64 --help

# For a Linux compatible binary
unzip fleet.zip 'linux/*' -d fleet
./fleet/linux/fleet_linux_amd64 --help
```

For more information on using Fleet, refer to the [Configuration](./2-Configuration.md) documentation.

## TLS configuration

In order for osqueryd clients to connect, the connection to Fleet must use TLS. The TLS connection may be terminated by Fleet itself, or by a proxy serving traffic to Fleet.

### TLS certificate considerations

- The CNAME or one of the Subject Alternate Names (SANs) on the certificate must match the hostname that osquery clients use to connect to the server/proxy.
- If self-signed certificates are used, the full certificate chain must be provided to osquery via the `--tls_server_certs` flag.
- If Fleet terminates TLS, consider using an ECDSA (rather than RSA) certificate, as RSA certificates have been associated with [performance problems in Fleet due to Go's standard library TLS implementation](https://github.com/fleetdm/fleet/issues/655).

## Infrastructure dependencies

Fleet currently has two infrastructure dependencies in addition to the `fleet` web server itself. Those dependencies are MySQL and Redis.

### MySQL

Fleet uses MySQL extensively as its main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker container](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `fleet` binary to use the correct MySQL instance, see the [Configuration](./2-Configuration.md) document.

Fleet requires at least MySQL version 5.7.

For host expiry configuration, the [event scheduler](https://dev.mysql.com/doc/refman/5.7/en/events-overview.html) must be enabled. This can be enabled via the command line, configuration file, or a user with the required privileges.

### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker container](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `fleet` binary to use the correct Redis instance, see the [Configuration](./2-Configuration.md) document.
