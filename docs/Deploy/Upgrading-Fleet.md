# Upgrading Fleet

This guide explains how to upgrade your Fleet instance to the latest version in order to get the latest features and bug fixes. For initial installation instructions, see [Installing Fleet](https://fleetdm.com/docs/deploy/deploy-fleet-on-centos#installing-fleet).

There are three steps to perform a typical Fleet upgrade:

1. [Installing the latest version](#install-the-latest-version-of-fleet)
2. [Preparing the database](#prepare-the-database)
3. [Serving the new Fleet instance](#serve-the-new-version)


## Install the latest version of Fleet

Fleet may be installed locally, or used in a Docker container. Follow the appropriate method for your environment. 

### Local installation

[Download](https://github.com/fleetdm/fleet/releases) the latest version of Fleet. Check the `Upgrading` section of the release notes for any additional steps that may need to be taken for a specific release. 

Unzip the newly downloaded version, and replace the existing Fleet version with the new, unzipped version.

For example, after downloading:

```sh
unzip fleet.zip 'linux/*' -d fleet
sudo cp fleet/linux/fleet* /usr/bin/
```

### Docker container

Pull the latest Fleet docker image:

```sh
docker pull fleetdm/fleet
```

## Prepare the database

Changes to Fleet may include changes to the database. Running the built-in database migrations will ensure that your database is set up properly for the currently installed version. 

It is always advised to [back up the database](https://dev.mysql.com/doc/refman/8.0/en/backup-methods.html) before running migrations. 

Database migrations in Fleet are intended to be run while the server is offline. Osquery is designed to be resilient to short downtime from the server, so no data will be lost from `osqueryd` clients in this process. Even on large Fleet installations, downtime during migrations is usually only seconds to minutes.

> First, take the existing servers offline.

Run database migrations:

```sh
fleet prepare db
```

## Serve the new version

Once Fleet has been replaced with the newest version and the database migrations have completed, serve the newly upgraded Fleet instance:

```sh
fleet serve
```

## AWS with Terraform

If you are using Fleet's Terraform modules to manage your Fleet deployment to AWS, update the version in `main.tf`:

```tf
  fleet_config = {
    image = "fleetdm/fleet:<version>" 
    [...]
  }
```

Run `terraform apply` to apply the changes.

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Learn how to upgrade your Fleet instance to the latest version.">
