package dev_table_tooling

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

// allowedCommand encapsulates the possible binary path(s) of a command allowed to execute
// along with a strict list of arguments.
type allowedCommand struct {
	binPaths []string
	args     []string
}

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("name"),
		table.TextColumn("args"),
		table.TextColumn("output"),
		table.TextColumn("error"),
	}

	tableName := "dev_table_tooling"

	t := &Table{
		logger: log.With(logger, "table", tableName),
	}

	return table.NewPlugin(tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	for _, name := range tablehelpers.GetConstraints(queryContext, "name", tablehelpers.WithDefaults("")) {
		if name == "" {
			level.Info(t.logger).Log("msg", "Command name must not be blank")
			continue
		}

		cmd, ok := allowedCommands[name]

		if !ok {
			level.Info(t.logger).Log("msg", "Command not allowed", "name", name)
			continue
		}

		result := map[string]string{
			"name":   name,
			"args":   strings.Join(cmd.args, " "),
			"output": "",
			"error":  "",
		}

		if output, err := tablehelpers.Exec(ctx, t.logger, 30, cmd.binPaths, cmd.args, false); err != nil {
			level.Info(t.logger).Log("msg", "execution failed", "name", name, "err", err)
			result["error"] = err.Error()
		} else {
			result["output"] = base64.StdEncoding.EncodeToString(output)
		}

		results = append(results, result)
	}

	return results, nil
}
