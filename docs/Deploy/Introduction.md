# Introduction

<!-- TODO: video -->

Fleet is the most widely used open source osquery manager in the world. Fleet enables programmable live queries, streaming logs, and realtime visibility of 100,000+ servers, containers, and laptops. It's especially useful for IT, security, and compliance use cases.

The Fleet application contains two single static binaries which provide web based administration, REST API, and CLI interface to Fleet.

The `fleet` binary contains:
- The Fleet TLS web server (no external webserver is required but it supports a proxy if desired)
- The Fleet web interface
- The Fleet application management [REST API](https://fleetdm.com/docs/using-fleet/rest-api)
- The Fleet osquery API endpoints

The `fleetctl` binary is the CLI interface which allows management of your deployment, scriptable live queries, and easy integration into your existing logging, alerting, reporting, and management infrastructure.

Both binaries are available for download from our [repo](https://github.com/fleetdm/fleet/releases).

## Fleet vs Fleet Sandbox

If you'd like to try Fleet on your laptop, we recommend [Fleet Sandbox](https://fleetdm.com/try-fleet/register).

If you want to enroll real hosts or deploy to a more scalable environment, we recommend [deploying Fleet to a server](https://fleetdm.com/docs/deploying/server-installation).

## Infrastructure dependencies

Fleet currently has three infrastructure dependencies: MySQL, Redis, and a TLS certificate.

![Fleet's architecture diagram](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/fleet-architecture-diagram.png)

### MySQL

Fleet uses MySQL extensively as its main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker image](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `fleet` binary to use the correct MySQL instance, see the [Configuration](https://fleetdm.com/docs/deploying/configuration) document.

Fleet requires at least MySQL version 5.7.

### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker image](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `fleet` binary to use the correct Redis instance, see the [Configuration](https://fleetdm.com/docs/deploying/configuration) document.

## TLS certificate

In order for osqueryd clients to connect, the connection to Fleet must use TLS. The TLS connection may be terminated by Fleet itself, or by a proxy serving traffic to Fleet.

- The CNAME or one of the Subject Alternate Names (SANs) on the certificate must match the hostname that osquery clients use to connect to the server/proxy.
- If you intend to have your Fleet instance on a subdomain, your certificate can have a wildcard SAN. So `fleet.example.com` should match a SAN of `*.example.com`
- If self-signed certificates are used, the full certificate chain must be provided to osquery via the `--tls_server_certs` flag.
- If Fleet terminates TLS, consider using an ECDSA (rather than RSA) certificate, as RSA certificates have been associated with [performance problems in Fleet due to Go's standard library TLS implementation](https://github.com/fleetdm/fleet/issues/655).

<meta name="pageOrderInSection" value="100">
<meta name="description" value="Learn about Fleet's architecture and infrastructure dependencies.">
