# Introduction

- [Overview](#overview)
- [Fleet vs Fleet Preview](#Fleet-vs-Fleet-Preview)
- [Infrastructure dependencies](#infrastructure-dependencies)
  - [MySQL](#mysql)
  - [Redis](#redis)
  - [TLS certificate](#tls-certificate)

The Fleet application is distributed as a single static binary (or as a Docker container). This binary serves:

Fleet is the most widely used open source osquery manager in the world. Fleet enables programmable live queries, streaming logs, and realtime visibility of 100,000+ servers, containers, and laptops. It's especially useful for IT, security, and compliance use cases.

The Fleet application contains two single static binaries which provide web based administration, REST API, and CLI interface to Fleet.

The `fleet` binary contains:
- The Fleet TLS web server (no external webserver is required but it supports a proxy if desired)
- The Fleet web interface
- The Fleet application management [REST API](../01-Using-Fleet/03-REST-API.md)
- The Fleet osquery API endpoints

The `fleetctl` binary is the CLI interface which allows management of your deployment, scriptable live queries, and easy integration into your existing logging, alerting, reporting, and management infrastructure.

Both binaries are available for download from our [repo](https://github.com/fleetdm/fleet/releases).

## Fleet vs Fleet Preview

If you'd like to try Fleet on your laptop we recommend [Fleet Preview](https://fleetdm.com/get-started): a convenient Docker instance that includes all infrastructure dependencies, sample virtual hosts, and the option to enroll your laptop for testing included.

If you want to enroll real hosts or deploy to a more scalable environment we recommend [deploying Fleet to a server](./02-Server-Installation.md).

## Infrastructure dependencies

Fleet currently has three infrastructure dependencies: MySQL, Redis, and a TLS certificate.

### MySQL

Fleet uses MySQL extensively as its main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker container](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `fleet` binary to use the correct MySQL instance, see the [Configuration](./03-Configuration.md) document.

Fleet requires at least MySQL version 5.7.

For host expiry configuration, the [event scheduler](https://dev.mysql.com/doc/refman/5.7/en/events-overview.html) must be enabled. This can be enabled via the command line, configuration file, or a user with the required privileges.

### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker container](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `fleet` binary to use the correct Redis instance, see the [Configuration](./03-Configuration.md) document.

## TLS certificate

In order for osqueryd clients to connect, the connection to Fleet must use TLS. The TLS connection may be terminated by Fleet itself, or by a proxy serving traffic to Fleet.

- The CNAME or one of the Subject Alternate Names (SANs) on the certificate must match the hostname that osquery clients use to connect to the server/proxy.
- If self-signed certificates are used, the full certificate chain must be provided to osquery via the `--tls_server_certs` flag.
- If Fleet terminates TLS, consider using an ECDSA (rather than RSA) certificate, as RSA certificates have been associated with [performance problems in Fleet due to Go's standard library TLS implementation](https://github.com/fleetdm/fleet/issues/655).
