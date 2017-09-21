Updating Fleet
==============

This guide explains how to update and run new versions of Fleet. For initial installation instructions, see [Installing Fleet](./installing-fleet.md).

There are two steps to perform a typical Fleet update. If any other steps are required, they will be noted in the release notes.

1. [Update the Fleet binary](#updating-the-fleet-binary)
2. [Run database migrations](#running-database-migrations)

As with any enterprise software update, it's a good idea to back up your MySQL data before updating Fleet.

## Updating the Fleet binary

Follow the binary update instructions corresponding to the original installation method used to install Fleet.

#### Kolide quickstart script

The quickstart script will automatically update and migrate Fleet when run. In the `kolide-quickstart` directory:

```
./demo.sh up
```

Step 2 is performed automatically, so no further action is necessary.

#### Docker container

Pull the latest Fleet docker image:

```
docker pull kolide/kolide
```

#### Debian Packages (Ubuntu, Debian)

Update Fleet through the Apt repository (the repository should have been added during initial install):

```
sudo apt-get update && sudo apt-get install kolide
```

#### Yum Packages (CentOS, RHEL, Amazon Linux)

Update Fleet through the Yum respository (the repository should have been added during initial install):

```
sudo yum update kolide
```

#### Raw binaries

Download the latest raw Fleet binaries:

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

Replace the existing Fleet binary with the newly unzipped binary.

## Running database migrations

Before running the updated server, perform necessary database migrations:

```
kolide prepare db
```

Note, if you would like to run this in a script, you can use the `--no-prompt` option to disable prompting before the migrations.

The updated Fleet server should now be ready to run:

```
kolide serve
```
