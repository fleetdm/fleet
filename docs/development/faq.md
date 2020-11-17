# Troubleshooting FAQ

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
server/kolide/emails.go:90:23: undefined: Asset
make: *** [fleet] Error 2
```

If you get an `undefined: Asset` error it is likely because you did not run `make generate` before `make build`. See [Building the Code](https://github.com/fleetdm/fleet/blob/master/docs/development/building-the-code.md) for additional documentation on compiling the `fleet` binary.

## How do I connect to the Mailhog simulated mail server?

First, in the Fleet UI, navigate to the App Settings page under Admin. Then, in the "SMTP Options" section, enter any email address in the "Sender Address" field, set the "SMTP Server" to `localhost` on port `1025`, and set "Authentication Type" to `None`. Note that you may use any active or inactive sender address.

Visit [locahost:8025](http://localhost:8025) to view Mailhog's admin interface which will display all emails sent using the simulated mail server.
