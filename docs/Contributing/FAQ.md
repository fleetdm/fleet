# Contribution FAQ

- [Make errors](#make-errors)
  - [`dep: command not found`](#dep-command-not-found)
  - [`undefined: Asset`](#undefined-asset)
- [How do I connect to the MailHog simulated mail server?](#how-do-i-connect-to-the-mailhog-simulated-mail-server)
- [Adding hosts for testing](#adding-hosts-for-testing)
- [Why am I getting an error about self-signed certificates when running `fleetctl preview`?](#why-am-i-getting-an-error-about-self-signed-certificates-when-running-fleetctl-preview)
- [Will updating fleetctl lead to loss of data in fleetctl preview?](#will-updating-fleetctl-lead-to-loss-of-data-in-fleetctl-preview?)


## Enrolling in multiple Fleet servers
Enrolling your device with more than one Fleet server is not currently possible.  Multiple install roots are useful for the development of Fleet itself but complex to maintain.  While this has some value for Fleet contributors, there is currently no active effort to add and maintain support for multiple enrollments from the same device.

## Make errors

### `dep: command not found`

```sh
/bin/bash: dep: command not found
make: *** [.deps] Error 127
```

If you get the above error, you need to add `$GOPATH/bin` to your PATH. A quick fix is to run `export PATH=$GOPATH/bin:$PATH`.
See the Go language documentation for [workspaces](https://golang.org/doc/code.html#Workspaces) and [GOPATH](https://golang.org/doc/code.html#GOPATH) for more in-depth documentation.

### `undefined: Asset`

```sh
server/fleet/emails.go:90:23: undefined: Asset
make: *** [fleet] Error 2
```

If you get an `undefined: Asset` error, it is likely because you did not run `make generate` before `make build`. See [Building Fleet](https://fleetdm.com/docs/contributing/building-fleet) for additional documentation on compiling the `fleet` binary.

## Adding hosts for testing

The `osquery` directory contains a docker-compose.yml and additional configuration files to start containerized osquery agents.

To start osquery, first retrieve the "Enroll secret" from Fleet (by clicking the "Add New Host") button in the Fleet dashboard, or with `fleetctl get enroll-secret`).

```sh
cd tools/osquery
ENROLL_SECRET=<copy from fleet> docker-compose up
```

## Why am I getting an error about self-signed certificates when running `fleetctl preview`?

If you are trying to run `fleetctl preview` and seeing errors about self-signed certificates, the
most likely culprit is that you're behind a corporate proxy server and need to [add the proxy
settings to Docker](https://docs.docker.com/network/proxy/) so that the container created by
`fleetctl preview` is able to connect properly.

## Will updating fleetctl lead to loss of data in fleetctl preview?

No, you won't experience data loss when you update fleetctl. Note that you can run `fleetctl preview --tag v#.#.#` if you want to run Preview on a previous version. Just replace # with the version numbers of interest.

## Can I disable usage statistics via the config file or a CLI flag?
Apart from an admin [disabling usage](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics) statistics on the Fleet UI, you can edit your `fleet.yml` config file to disable usage statistics. Look for the `server_settings` in your `fleet.yml` and set `enable_analytics: false`. Do note there is no CLI flag option to disable usage statistics at this time.

## Fleet preview fails with Invalid interpolation. What should I do?

If you tried running `fleetctl preview` and you get the following error:

```sh
fleetctl preview
Downloading dependencies into /root/.fleet/preview...
Pulling Docker dependencies...
Invalid interpolation format for "fleet01" option in service "services": "fleetdm/fleet:${FLEET_VERSION:-latest}"

Failed to run docker-compose
```

You're probably running an old version of Docker. Download the installer for your platform from the [Docker Documentation](https://docs.docker.com/compose/install/).

## What API endpoints do osquery and Fleetd need access to?

Based on the configuration, osquery running on hosts will need access to these API endpoints:

* `/api/v1/osquery/enroll`
* `/api/v1/osquery/config`
* `/api/v1/osquery/distributed/read`
* `/api/v1/osquery/distributed/write`
* `/api/v1/osquery/carve/begin`
* `/api/v1/osquery/carve/block`
* `/api/v1/osquery/log`

If you also have Fleetd running on hosts, it will need access to these API endpoints:

* `/api/fleet/orbit/enroll`
* `/api/fleet/orbit/config`
* `/api/fleet/orbit/device_token`
* `/api/fleet/orbit/ping`
* `/api/fleet/orbit/scripts/request`
* `/api/fleet/orbit/scripts/result`
* `/api/fleet/orbit/disk_encryption_key`
* `/api/fleet/orbit/device_mapping`
* `/api/osquery/log`

Hosts running Fleet Desktop will need access to these API endpoints:

* `/api/latest/fleet/device/.+/desktop`
* `/api/latest/fleet/device/.+/ping`

> Full list [here](https://github.com/fleetdm/fleet/blob/c080a3b0e1eed2184b4b7bb77a6abd8c2c39b9f4/server/service/handler.go#L791-L839)

<meta name="description" value="Find commonly asked questions and answers about contributing to Fleet as part of our community.">
