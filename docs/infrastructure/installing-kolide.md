Installing Kolide
=================

The Kolide application is distributed as a single static binary. This binary serves:

- The Kolide web interface
- The Kolide application API endpoints
- The osquery TLS server API endpoints

All of these are served via a built-in HTTP server, so there is no need for complex web server configurations. Once you've installed the `kolide` binary and it's infrastructure dependencies as illustrated below, refer to the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) documentation for information on how to use and configure the Kolide application.

## Kolide Quickstart

Kolide provides a [quickstart script](https://github.com/kolide/kolide-quickstart) that is the quickest way to get a demo Kolide instance up and running. For easiest install, see the instructions provided on your [Kolide license page](https://kolide.co/account/product-and-license#quick-start).

Note that the quickstart is not intended to be a production deployment. If you would like a production deployment, please choose one of the methods below.

## Installing the Kolide binary

Because everyone's infrastructure is different, there are a multiple options available for installing the Kolide binary.

#### Docker container

Pull the latest Kolide docker image:

```
docker pull kolide/kolide
```

For more information on using Kolide, refer to the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) documentation.

#### Debian Packages (Ubuntu, Debian)

Add our GPG key and install the Kolide Apt Repository:

```
wget -qO - https://dl.kolide.co/archive.key | sudo apt-key add -
sudo add-apt-repository "deb https://dl.kolide.co/apt jessie main"
sudo apt-get update
```

Install Kolide:

```
sudo apt-get install kolide
/usr/bin/kolide --help
```

For more information on using Kolide, refer to the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) documentation.

#### Yum Packages (CentOS, RHEL, Amazon Linux)

Install the Kolide Yum Repository:

```
sudo rpm -ivh https://dl.kolide.co/yum/kolide-yum-repo-1.0.0-1.noarch.rpm
```

Install Kolide:

```
sudo yum install kolide
kolide --help
```

For more information on using Kolide, refer to the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) documentation.

#### Raw binaries

Download the latest raw Kolide binaries:

```
curl -O https://dl.kolide.co/bin/kolide_latest.zip
```

Unzip the binaries for your platform:

```
# For a Darwin compatible binary
unzip kolide_latest.zip 'darwin/*' -d kolide
./kolide/darwin/kolide_darwin_amd64 --help

# For a Linux compatible binary
unzip kolide_latest.zip 'linux/*' -d kolide
./kolide/linux/kolide_linux_amd64 --help
```

For more information on using Kolide, refer to the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) documentation.

## Infrastructure Dependencies

Kolide currently has two infrastructure dependencies in addition to the `kolide` web server itself. Those dependencies are MySQL and Redis.

#### MySQL

Kolide uses MySQL extensively as it's main database. Many cloud providers (such as [AWS](https://aws.amazon.com/rds/mysql/) and [GCP](https://cloud.google.com/sql/)) host reliable MySQL services which you may consider for this purpose. A well supported MySQL [Docker container](https://hub.docker.com/_/mysql/) also exists if you would rather run MySQL in a container. For more information on how to configure the `kolide` binary to use the correct MySQL instance, see the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) document.

#### Redis

Kolide uses Redis to ingest and queue the results of distributed queries, cache data, etc. Many cloud providers (such as [AWS](https://aws.amazon.com/elasticache/) and [GCP](https://console.cloud.google.com/launcher/details/click-to-deploy-images/redis)) host reliable Redis services which you may consider for this purpose. A well supported Redis [Docker container](https://hub.docker.com/_/redis/) also exists if you would rather run Redis in a container. For more information on how to configure the `kolide` binary to use the correct Redis instance, see the [Configuring The Kolide Binary](./configuring-the-kolide-binary.md) document.
