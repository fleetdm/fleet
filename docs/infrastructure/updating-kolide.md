Updating Kolide
=================

This guide explains how to update and run new versions of Kolide. For initial installation instructions, see [Installing Kolide](./installing-kolide.md).

There are two steps to perform a typical Kolide update. If any other steps are required, they will be noted in the release notes.

1. [Update the Kolide binary](#updating-the-kolide-binary)
2. [Run database migrations](#running-database-migrations)

As with any enterprise software update, it's a good idea to back up your MySQL data before updating Kolide.

## Updating the Kolide binary

Follow the binary update instructions corresponding to the original installation method used to install Kolide.

#### Kolide quickstart script

The quickstart script will automatically update and migrate Kolide when run. In the `kolide-quickstart` directory:

```
./demo.sh up
```

Step 2 is performed automatically, so no further action is necessary.

#### Docker container

Pull the latest Kolide docker image:

```
docker pull kolide/kolide
```

#### Debian Packages (Ubuntu, Debian)

Update Kolide through the Apt repository (the repository should have been added during initial install):

```
sudo apt-get update && sudo apt-get install kolide
```

#### Yum Packages (CentOS, RHEL, Amazon Linux)

Update Kolide through the Yum respository (the repository should have been added during initial install):

```
sudo yum update kolide
```

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

Replace the existing Kolide binary with the newly unzipped binary.

## Running database migrations

Before running the updated server, perform necessary database migrations:

```
kolide prepare db
```

The updated Kolide server should now be ready to run:

```
kolide serve
```
