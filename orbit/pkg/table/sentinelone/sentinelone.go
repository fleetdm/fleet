// Package sentinelone exposes the local SentinelOne agent's state, as reported
// by `sentinelctl status`, as the `sentinelone` osquery table.
package sentinelone

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// execTimeout is the maximum time to wait for a single sentinelctl invocation.
const execTimeout = 15 * time.Second

// errCLINotFound is returned when sentinelctl isn't installed where the
// SentinelOne installer puts it.
var errCLINotFound = errors.New("sentinelctl not found")

// runSentinelctl executes `sentinelctl <args...>` and returns its combined
// stdout/stderr. It is a variable so tests can inject fixture output without a
// SentinelOne install.
//
// Only the absolute paths the installer uses are executed. osquery runs this as
// root, so resolving the binary through $PATH would let anyone who can write to
// a directory on it run code as root.
var runSentinelctl = func(ctx context.Context, args ...string) ([]byte, error) {
	path := resolveCLIPath()
	if path == "" {
		return nil, errCLINotFound
	}

	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	return exec.CommandContext(ctx, path, args...).CombinedOutput()
}

// Columns returns the schema of the sentinelone table. Every column is TEXT:
// sentinelctl reports free-form status strings, and an empty string keeps
// "not reported on this platform" distinguishable from a real zero value.
func Columns() []table.ColumnDefinition {
	cols := make([]table.ColumnDefinition, 0, len(columnOrder))
	for _, name := range columnOrder {
		cols = append(cols, table.TextColumn(name))
	}
	return cols
}

// Generate returns a single row describing the local SentinelOne agent, or no
// rows when the agent is not installed or `sentinelctl status` reported nothing
// recognizable. It never returns an error, so `SELECT * FROM sentinelone`
// succeeds on hosts without SentinelOne.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	out, err := runSentinelctl(ctx, "status")
	if err != nil {
		log.Debug().Err(err).Msg("sentinelctl status failed; reporting no sentinelone rows")
		return []map[string]string{}, nil
	}

	parsed := parseStatus(string(out))
	applyWindowsCanonicalPaths(parsed)

	row := make(map[string]string, len(columnOrder))
	for _, name := range columnOrder {
		row[name] = ""
	}

	populated := false
	for path, val := range parsed {
		col, ok := columnPathMap[path]
		if !ok {
			continue
		}
		row[col] = val
		populated = true
	}
	if !populated {
		log.Debug().Msg("sentinelctl status had no recognized fields; reporting no sentinelone rows")
		return []map[string]string{}, nil
	}

	// Normalize timestamps to Unix epoch seconds. A value that won't parse is
	// cleared rather than passed through, so a numeric comparison in SQL can't
	// silently match a locale-formatted date.
	for _, col := range epochColumns {
		v := row[col]
		if v == "" {
			continue
		}
		if epoch, ok := toUnixEpoch(v); ok {
			row[col] = epoch
		} else {
			log.Debug().Str("column", col).Msg("unparseable sentinelctl timestamp; clearing column")
			row[col] = ""
		}
	}

	return []map[string]string{row}, nil
}
