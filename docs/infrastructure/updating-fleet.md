Updating Fleet
==============

This guide explains how to update and run new versions of Fleet. For initial installation instructions, see [Installing Fleet](./installing-fleet.md).

There are two steps to perform a typical Fleet update. If any other steps are required, they will be noted in the release notes.

1. [Update the Fleet binary](#updating-the-fleet-binary)
2. [Run database migrations](#running-database-migrations)

As with any enterprise software update, it's a good idea to back up your MySQL data before updating Fleet.

## Updating the Fleet binary

Follow the binary update instructions corresponding to the original installation method used to install Fleet.

#### Raw binaries

Download the latest raw Fleet binaries:

```
curl -O https://github.com/kolide/fleet/releases/latest/download/fleet.zip
```

Unzip the binaries for your platform:

```
# For a Darwin compatible binary
unzip fleet.zip 'darwin/*' -d fleet
./fleet/darwin/fleet --help

# For a Linux compatible binary
unzip fleet.zip 'linux/*' -d fleet
./fleet/linux/fleet --help
```

Replace the existing Fleet binary with the newly unzipped binary.

#### Docker container

Pull the latest Fleet docker image:

```
docker pull kolide/fleet
```

## Running database migrations

Before running the updated server, perform necessary database migrations:

```
fleet prepare db
```

Note, if you would like to run this in a script, you can use the `--no-prompt` option to disable prompting before the migrations.

The updated Fleet server should now be ready to run:

```
fleet serve
```
