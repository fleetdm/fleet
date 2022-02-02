# Contribution FAQ

- [Make errors](#make-errors)
  - [`dep: command not found`](#dep-command-not-found)
  - [`undefined: Asset`](#undefined-asset)
- [How do I connect to the Mailhog simulated mail server?](#how-do-i-connect-to-the-mailhog-simulated-mail-server)
- [Adding hosts for testing](#adding-hosts-for-testing)


## Enrolling in multiple Fleet servers

It is not currently possible to enroll your device with more than one Fleet server.  Multiple install roots is useful for development of Fleet itself, but complex to maintain.  While this has some value for Fleet contributors, there is currently no active effort to add and maintain support for multiple enrollments from the same device.

## Make errors

### `dep: command not found`

```
/bin/bash: dep: command not found
make: *** [.deps] Error 127
```

If you get the above error, you need to add `$GOPATH/bin` to your PATH. A quick fix is to run `export PATH=$GOPATH/bin:$PATH`.
See the Go language documentation for [workspaces](https://golang.org/doc/code.html#Workspaces) and [GOPATH](https://golang.org/doc/code.html#GOPATH) for a more indepth documentation.

### `undefined: Asset`

```
server/fleet/emails.go:90:23: undefined: Asset
make: *** [fleet] Error 2
```

If you get an `undefined: Asset` error it is likely because you did not run `make generate` before `make build`. See [Building Fleet](./Building-Fleet.md) for additional documentation on compiling the `fleet` binary.

## Adding hosts for testing

The `osquery` directory contains a docker-compose.yml and additional configuration files to start containerized osquery agents.

To start osquery, first retrieve the "Enroll secret" from Fleet (by clicking the "Add New Host") button in the Fleet dashboard, or with `fleetctl get enroll-secret`).

```
cd tools/osquery
ENROLL_SECRET=<copy from fleet> docker-compose up
```
