# NanoMDM

> The contents of this directory were copied (on January 2024) from https://github.com/fleetdm/nanomdm (the `apple-mdm` branch) which was forked from https://github.com/micromdm/nanomdm.
> They were updated in November 2024 with changes up to github.com/micromdm/nanomdm@825f2979a2dc28c6cc57bb62aff16737978bd90e

NanoMDM is a minimalist [Apple MDM server](https://developer.apple.com/documentation/devicemanagement) heavily inspired by [MicroMDM](https://github.com/micromdm/micromdm).

## Getting started & Documentation

- [Quickstart](docs/quickstart.md)
A quick guide to get NanoMDM up and running using ngrok.

- [Operations Guide](docs/operations-guide.md)
A brief overview of the various command-line switches and HTTP endpoints and APIs available to NanoMDM.

## Features

- Horizontal scaling: zero/minimal local state. Persistence in storage layers. MySQL and PostgreSQL backends provided in the box.
- Multiple APNs topics: potentially multi-tenant.
- Multi-command targeting: send the same command (or pushes) to multiple enrollments without individually queuing commands.
- Migration endpoint: allow migrating MDM enrollments between storage backends or (supported) MDM servers
- Otherwise we share many features between MicroMDM and NanoMDM, such as:
  - A MicroMDM-emulating HTTP webhook/callback.
  - Enrollment-certificate authorization
  - API-driven interaction (queuing of commands, APNs pushes, etc.)

## $x not included

NanoMDM is but one component for a functioning MDM server. At a minimum you need a SCEP server and TLS termination, for example. If you've used [MicroMDM](https://github.com/micromdm/micromdm) before you might be interested to know what NanoMDM does *not* include, by way of comparison.

- SCEP.
  - Spin up your own [scep](https://github.com/micromdm/scep) server. Or bring your own.
- TLS.
  - You'll need to provide your own reverse proxy/load balancer that terminates TLS.
- ADE (DEP) API access.
  - While ADE/DEP *enrollments* are supported there is no DEP API access.
- Enrollment (Profiles).
  - You'll need to create and serve your own enrollment profiles to devices.
- Blueprints.
  - No 'automatic' command sending upon enrollment. Entirely driven by webhook or other integrations.
- JSON command API.
  - Commands are submitted in raw Plist form only. See the [cmdr.py tool](tools/cmdr.py) that helps generate raw commands
  - The [micro2nano](https://github.com/micromdm/micro2nano) project provides an API translation server between MicroMDM's JSON command API and NanoMDM's raw Plist API.
- VPP.
- Enrollment (device) APIs.
  - No ability, yet, to inspect enrollment details or state.
  - This is partly mitigated by the fact that both the `file` and `mysql` storage backends are "easy" to inspect and query.

## Architecture Overview

NanoMDM, at its core, is a thin composable layer between HTTP handlers and a set of storage abstractions.

- The "front-end" is a set of standard Golang HTTP handlers that handle MDM and API requests. The core MDM handlers adapt the requests to the service layer. These handlers exist in the `http` package.
- The service layer is a composable interface for processing and handling MDM requests. The main NanoMDM service dispatches to the storage layer. These services exist under the `service` package.
- The storage layer is a set of interfaces and implementations that store & retrieve MDM enrollment and command data. These exist under the `storage` package.

You can read more about the architecture in the blog post [Introducing NanoMDM](https://micromdm.io/blog/introducing-nanomdm/).

## Running unit tests locally

1. Start up MySQL `docker compose up`
2. Load schema: `mysql --user=fleet -pinsecure --host=127.0.0.1 --port=3800 --protocol=TCP fleet < ./storage/mysql/schema.sql`
3. `export NANOMDM_MYSQL_STORAGE_TEST_DSN=fleet:insecure@tcp(127.0.0.1:3800)/fleet`
4. `go test -v -parallel 8 -race=true ./...`

### Clean up MySQL after running tests

```mysql
SET FOREIGN_KEY_CHECKS = 0;
truncate table nano_cert_auth_associations;
truncate table nano_command_results;
truncate table nano_commands;
truncate table nano_devices;
truncate table nano_enrollment_queue;
truncate table nano_enrollments;
truncate table nano_push_certs;
truncate table nano_users;
SET FOREIGN_KEY_CHECKS = 1;
```
