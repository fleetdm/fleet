Installing Kolide
=================

The Kolide application is distributed as a single static binary. This binary serves:

- The Kolide web interface
- The Kolide application API endpoints
- The osquery TLS server API endpoints

All of these are served via a built-in HTTP server, so there is no need for complex web server configurations. Once you've installed the `kolide` binary and it's infrastructure dependencies as illustrated below, refer to the [Running Kolide](./running-kolide.md) documentation for information on how to use and configure the Kolide application.

## Installing the Kolide binary

Because everyone's infrastructure is different, there are a multiple options available for installing the Kolide binary.

#### Docker container

If you'd like to run the Kolide server from a supported Docker container, we currently build and publish Docker containers via our CI process for every commit to master. We then run each of these Docker containers as apart of our testing process. At this time, please contact [support@kolide.co](mailto:support@kolide.co) and we will add you to our Docker Hub account so that you can access the private container images. When the Kolide product is out of beta, these containers will be made public.

#### Debian Packages (Ubuntu, Debian)

We are currently working on an apt repository where we will upload deb packages of the Kolide application in both a stable and "master" channel. Please check back here for information on this feature.

#### Yum Packages (CentOS, RHEL, Amazon Linux)

We are currently working on a yum repository where we will upload rpm packages of the Kolide application in both a stable and "master" channel. Please check back here for information on this feature.

#### Raw binaries

The `kolide` binary is a single statically linked binary with no dependencies. For users that would like to download the raw binaries for more custom internal distribution, we will be hosting the raw binaries to support this use-case. Please check back here for information on this feature.

## Infrastructure Dependencies

Kolide currently has two infrastructure dependencies in addition to the `kolide` web server itself. Those dependencies are MySQL and Redis.

#### MySQL

Kolide uses MySQL extensively as it's main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker container](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `kolide` binary to use the correct MySQL instance, see the [Running Kolide](./running-kolide.md) document.

#### Redis

Kolide uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker container](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `kolide` binary to use the correct Redis instance, see the [Running Kolide](./running-kolide.md) document.