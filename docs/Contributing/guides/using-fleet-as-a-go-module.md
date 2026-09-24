# Using Fleet as a Go module

Fleet's server code is a valid Go module (`github.com/fleetdm/fleet/v4`), so you can import its packages into your own Go projects — for example, to build tooling on top of Fleet's API client or reuse its types.

## The Go module proxy doesn't index Fleet

By default `go get` fetches modules through the public Go module proxy (`proxy.golang.org`), which also powers [pkg.go.dev](https://pkg.go.dev). Fleet is **not** available this way.

To serve a module, the proxy does a shallow `git fetch` of the requested tag and builds a module zip from the tracked tree at that tag. Fleet is a large monorepo (~600 MiB of tracked files at a release tag, dominated by `website/`, `assets/`, and other non-Go directories), so the fetch and zip build consistently exceed the proxy's per-request deadline. The request times out and gets negative-cached, so recent versions never appear on pkg.go.dev.

This is a limitation of the proxy handling a large repo — the module itself is valid and builds fine. It just needs to be fetched directly from GitHub.

## Fetch directly from GitHub with GOPRIVATE

Set [`GOPRIVATE`](https://go.dev/ref/mod#private-modules) so the Go toolchain bypasses the proxy for Fleet and clones directly from GitHub instead:

```bash
GOPRIVATE=github.com/fleetdm/fleet go get github.com/fleetdm/fleet/v4@v4.83.0
```

Replace `v4.83.0` with the release version you want to pin to. You can also set it in your environment so you don't have to prefix every command:

```bash
go env -w GOPRIVATE=github.com/fleetdm/fleet
go get github.com/fleetdm/fleet/v4@v4.83.0
```

Once it's added to your `go.mod`, import packages as usual:

```go
package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func main() {
	var h fleet.Host
	fmt.Println(h.DisplayName())
}
```

Then build:

```bash
go build ./...
```

## Notes

- Fetching directly from GitHub is slower than the proxy (Go clones the repo), but it's a one-time cost per version.
- Because Fleet isn't indexed on pkg.go.dev, browse the package documentation and source in-repo instead.
- `GOPRIVATE` also disables checksum-database verification for the matched path. The download still records a hash in your `go.sum`, so subsequent builds remain reproducible.
