package mcp_servers

import (
	"context"
	"fmt"
	"time"

	osqclient "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

// osqClient abstracts the osquery client for ease of testing.
type osqClient interface {
	QueryRowContext(ctx context.Context, sql string) (map[string]string, error)
	QueryRowsContext(ctx context.Context, sql string) ([]map[string]string, error)
	Close()
}

var newClient = func(socket string, timeout time.Duration) (osqClient, error) {
	return osqclient.NewClient(socket, timeout)
}

// Columns defines the schema for the mcp_servers table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("pid"),
		table.TextColumn("name"),
		table.TextColumn("cmdline"),
	}
}

// Generate connects to the running osqueryd over the provided socket and queries
// the core processes table to list running processes. It supports constraint
// pushdown for pid (equals) and name (LIKE pattern).
func Generate(ctx context.Context, queryContext table.QueryContext, socket string) ([]map[string]string, error) {
	// Ensure we don't hang forever if osquery is unresponsive.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Open an osquery client using the extension socket.
	c, err := newClient(socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("open osquery client: %w", err)
	}
	defer c.Close()

	// First get the running processes with listening ports
	sql := `SELECT DISTINCT lp.pid, lp.port, lp.protocol, lp.family, lp.address, lp.path, p.name, p.path, p.cmdline from listening_ports lp CROSS JOIN processes p USING (pid)`

	rows, err := c.QueryRowsContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
