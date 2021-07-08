# Updating Fleet
- [Overview](#overview)
- [Updating the Fleet binary](#updating-the-fleet-binary)
  - [Raw binaries](#raw-binaries)
  - [Docker container](#docker-container)
- [Running database migrations](#running-database-migrations)

## Overview

This guide explains how to update and run new versions of Fleet. For initial installation instructions, see [Installing Fleet](../2-Deploying/1-Installation.md).

There are two steps to perform a typical Fleet update. If any other steps are required, they will be noted in the release notes.

1. [Update the Fleet binary](#updating-the-fleet-binary)
2. [Run database migrations](#running-database-migrations)

As with any enterprise software update, it's a good idea to back up your MySQL data before updating Fleet.

## Updating the Fleet binary

To update to a new version of Fleet, follow the [same binary install instructions](https://github.com/fleetdm/fleet/blob/main/docs/2-Deploying/1-Installation.md) from the original installation method you used to install Fleet.

### Raw binaries

Download, unzip, and replace the existing Fleet binary with the newly unzipped binary.

### Docker container

Pull the latest Fleet docker image:

```
docker pull fleetdm/fleet
```

## Running database migrations

Before running the updated server, perform necessary database migrations. It is always advised to back up the database before running migrations.

Database migrations in Fleet are intended to be run while the server is offline. Osquery is designed to be resilient to short downtime from the server, so no data will be lost from `osqueryd` clients in this process. Even on large Fleet installations, downtime during migrations is usually only seconds to minutes.

First, take the existing servers offline.

Run database migrations:

```
fleet prepare db
```

Note, if you would like to run this in a script, you can use the `--no-prompt` option to disable prompting before the migrations.

Start new Fleet server instances:

```
fleet serve
```
