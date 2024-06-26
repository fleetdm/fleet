# NanoMDM

> The contents of this directory were copied (on January 2024) from https://github.com/fleetdm/nanomdm (the `apple-mdm` branch) which was forked from https://github.com/micromdm/nanomdm.

[![Go](https://github.com/micromdm/nanomdm/workflows/Go/badge.svg)](https://github.com/micromdm/nanomdm/actions)

NanoMDM is a minimalist [Apple MDM server](https://support.apple.com/business/enrollment-deployment) heavily inspired by MicroMDM.

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
