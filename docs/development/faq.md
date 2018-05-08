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

If you get an `undefined: Asset` error it is likely because you did not run `make generate` before `make build`. See [Building the Code](https://github.com/kolide/fleet/blob/master/docs/development/building-the-code.md) for additional documentation on compiling the `fleet` binary.
