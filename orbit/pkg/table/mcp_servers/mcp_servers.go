package mcp_servers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	sql, singleRow, err := buildSQL(queryContext)
	if err != nil {
		return nil, err
	}

	if singleRow {
		row, err := c.QueryRowContext(ctx, sql)
		if err != nil {
			return nil, err
		}
		if row == nil {
			return nil, nil
		}
		return []map[string]string{row}, nil
	}

	rows, err := c.QueryRowsContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// buildSQL constructs the SQL query string for the processes table based on
// constraints in the query context. It returns the SQL string and whether the
// query is constrained to a single pid (singleRow=true).
func buildSQL(queryContext table.QueryContext) (string, bool, error) {
	var whereClauses []string
	singleRow := false

	// Handle pid equality constraint for single-row optimization.
	if cons, ok := queryContext.Constraints["pid"]; ok {
		for _, c := range cons.Constraints {
			if c.Operator == table.OperatorEquals {
				// Accept only valid integers for pid to avoid SQL injection.
				if _, err := strconv.ParseInt(c.Expression, 10, 64); err != nil {
					continue
				}
				whereClauses = append(whereClauses, "pid = "+c.Expression)
				singleRow = true
				break
			}
		}
	}

	// Handle name LIKE constraint(s).
	if cons, ok := queryContext.Constraints["name"]; ok {
		var likes []string
		for _, c := range cons.Constraints {
			if c.Operator == table.OperatorLike || c.Operator == table.OperatorEquals {
				pattern := escapeSQLString(c.Expression)
				likes = append(likes, "name LIKE '"+pattern+"'")
			}
		}
		if len(likes) > 0 {
			// Combine multiple LIKEs with OR, and group them.
			whereClauses = append(whereClauses, "("+strings.Join(likes, " OR ")+")")
			// If singleRow wasn't already true due to pid, keep as is. Name LIKE may match multiple.
		}
	}

	// Build final SQL.
	base := "SELECT pid, name, cmdline FROM processes"
	if len(whereClauses) > 0 {
		base += " WHERE " + strings.Join(whereClauses, " AND ")
	} else {
		// Defensive cap to avoid extremely large responses when unconstrained.
		base += " LIMIT 5000"
	}
	return base, singleRow, nil
}

// escapeSQLString escapes single quotes for inclusion in a single-quoted SQL literal.
func escapeSQLString(in string) string {
	return strings.ReplaceAll(in, "'", "''")
}
