Kolide Documentation
====================

Welcome to the Kolide documentation.

- Information about using the Kolide web application can be found in the [Application Documentation](./application/README.md).
- If you're interested in using the new `fleetctl` CLI to manage your osquery fleet, see the [CLI Documentation](./cli/README.md).
- Resources for deploying osquery to hosts, deploying the Kolide server, installing Kolide's infrastructure dependencies, etc. can all be found in the [Infrastructure Documentation](./infrastructure/README.md).
- If you are interested in accessing the Kolide REST API in order to programmatically interact with your osquery installation, please see the [API Documentation](./api/README.md).
- Finally, if you're interested in interacting with the Kolide source code, you will find information on modifying and building the code in the [Development Documentation](./development/README.md).

If you have any questions, please don't hesitate to [File a GitHub issue](https://github.com/kolide/fleet/issues) or [join us on Slack](https://osquery-slack.herokuapp.com/). You can find us in the `#kolide` channel.

# Troubleshooting FAQ

## Make errors

```
/bin/bash: dep: command not found
make: *** [.deps] Error 127
```

If you get the above error, you need to add `$GOPATH/bin` to your PATH. A quick fix is to run `export PATH=$GOPATH/bin:$PATH`. 
See the Go language documentation for [workspaces](https://golang.org/doc/code.html#Workspaces) and [GOPATH](https://golang.org/doc/code.html#GOPATH) for a more indepth documentation.

```
server/kolide/emails.go:90:23: undefined: Asset
make: *** [fleet] Error 2
```

If you get an `undefined: Asset` error it is likely because you did not run `make generate` before `make build`. See [Building the Code](https://github.com/kolide/fleet/blob/master/docs/development/building-the-code.md) for additional documentation on compiling the `fleet` binary.
