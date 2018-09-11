Installing Fleet
================

The Fleet application is distributed as a single static binary. This binary serves:

- The Fleet web interface
- The Fleet application API endpoints
- The osquery TLS server API endpoints

All of these are served via a built-in HTTP server, so there is no need for complex web server configurations. Once you've installed the `fleet` binary and it's infrastructure dependencies as illustrated below, refer to the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) documentation for information on how to use and configure the Fleet application.

## Installing the Fleet binary

Because everyone's infrastructure is different, there are a multiple options available for installing the Fleet binary.

#### Docker container

Pull the latest Fleet docker image:

```
docker pull kolide/fleet
```

For more information on using Fleet, refer to the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) documentation.

#### Raw binaries

Download the latest raw Fleet binaries:

```
curl -O https://dl.kolide.co/bin/fleet_latest.zip
```

Unzip the binaries for your platform:

```
# For a Darwin compatible binary
unzip fleet_latest.zip 'darwin/*' -d fleet
./fleet/darwin/fleet_darwin_amd64 --help

# For a Linux compatible binary
unzip fleet_latest.zip 'linux/*' -d fleet
./fleet/linux/fleet_linux_amd64 --help
```

For more information on using Fleet, refer to the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) documentation.

## Infrastructure Dependencies

Fleet currently has two infrastructure dependencies in addition to the `fleet` web server itself. Those dependencies are MySQL and Redis.

#### MySQL

Fleet uses MySQL extensively as it's main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker container](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `fleet` binary to use the correct MySQL instance, see the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) document.

Fleet requires at least MySQL version 5.7.

#### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker container](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `fleet` binary to use the correct Redis instance, see the [Configuring The Fleet Binary](./configuring-the-fleet-binary.md) document.
