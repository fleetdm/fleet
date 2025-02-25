// based on github.com/kolide/launcher/pkg/osquery/tables
package cryptsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

var cryptsetupPaths = []string{
	"/usr/sbin/cryptsetup",
	"/sbin/cryptsetup",
}

const allowedNameCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/_"

type Table struct {
	logger zerolog.Logger
	name   string
}

func TablePlugin(logger zerolog.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("name"),
	)

	t := &Table{
		logger: logger.With().Str("table", "cryptsetup_status").Logger(),
		name:   "cryptsetup_status",
	}

	return table.NewPlugin(t.name, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedNames := tablehelpers.GetConstraints(queryContext, "name",
		tablehelpers.WithAllowedCharacters(allowedNameCharacters),
		tablehelpers.WithLogger(t.logger),
	)

	if len(requestedNames) == 0 {
		return results, fmt.Errorf("The %s table requires that you specify a constraint for name", t.name)
	}

	for _, name := range requestedNames {
		output, err := tablehelpers.Exec(ctx, t.logger, 15, cryptsetupPaths, []string{"--readonly", "status", name}, false)
		if err != nil {
			t.logger.Debug().Err(err).Str("name", name).Msg("Error execing for status")
			continue
		}

		status, err := parseStatus(output)
		if err != nil {
			t.logger.Info().Err(err).Str("name", name).Msg("Error parsing status")
			continue
		}

		for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
			flatData, err := t.flattenOutput(dataQuery, status)
			if err != nil {
				t.logger.Info().Err(err).Msg("flatten failed")
				continue
			}

			rowData := map[string]string{"name": name}

			results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
		}
	}

	return results, nil
}

func (t *Table) flattenOutput(dataQuery string, status map[string]interface{}) ([]dataflatten.Row, error) {
	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	return dataflatten.Flatten(status, flattenOpts...)
}
