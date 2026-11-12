//go:build linux

package containerd

import (
	"github.com/containerd/containerd"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	defaultSocketPath = "/run/containerd/containerd.sock"
	socketPathCol     = "socket_path"
)

// resolveSocketPath fetches socket path from the query context.
func resolveSocketPath(queryContext table.QueryContext) string {
	paths := tablehelpers.GetConstraints(queryContext, socketPathCol, tablehelpers.WithDefaults(defaultSocketPath))
	if len(paths) == 0 {
		return defaultSocketPath
	}
	return paths[0]
}

// newClient wraps the creation of containerd.Client to handle the socket path.
func newClient(queryContext table.QueryContext) (*containerd.Client, string, error) {
	sp := resolveSocketPath(queryContext)
	client, err := containerd.New(sp)
	return client, sp, err
}
