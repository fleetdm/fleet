// based on github.com/kolide/launcher/pkg/osquery/tables
package cryptoinfotable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/cryptoinfo"
	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

type Table struct {
	logger zerolog.Logger
}

func TablePlugin(logger zerolog.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("passphrase"),
		table.TextColumn("path"),
	)

	t := &Table{
		logger: logger.With().Str("table", "cryptoinfo").Logger(),
	}

	return table.NewPlugin("cryptoinfo", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedPaths := tablehelpers.GetConstraints(queryContext, "path")
	if len(requestedPaths) == 0 {
		return results, errors.New("The cryptoinfo table requires that you specify an equals constraint for path")
	}

	for _, requestedPath := range requestedPaths {

		// We take globs in via the sql %, but glob needs *. So convert.
		filePaths, err := filepath.Glob(strings.ReplaceAll(requestedPath, `%`, `*`))
		if err != nil {
			t.logger.Info().Err(err).Msg("bad file glob")
			continue
		}

		for _, filePath := range filePaths {
			for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
				for _, passphrase := range tablehelpers.GetConstraints(queryContext, "passphrase", tablehelpers.WithDefaults("")) {

					flattenOpts := []dataflatten.FlattenOpts{
						dataflatten.WithLogger(t.logger),
						dataflatten.WithNestedPlist(),
						dataflatten.WithQuery(strings.Split(dataQuery, "/")),
					}

					flatData, err := flattenCryptoInfo(filePath, passphrase, flattenOpts...)
					if err != nil {
						t.logger.Info().Err(err).Str("path", filePath).Msg("failed to get data for path")
						continue
					}

					rowData := map[string]string{
						"path":       filePath,
						"passphrase": passphrase,
					}
					results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)

				}
			}
		}
	}
	return results, nil
}

// flattenCryptoInfo is a small wrapper over pkg/cryptoinfo that passes it off to dataflatten for table generation
func flattenCryptoInfo(filename, passphrase string, opts ...dataflatten.FlattenOpts) ([]dataflatten.Row, error) {
	filebytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}

	result, err := cryptoinfo.Identify(filebytes, passphrase)
	if err != nil {
		return nil, fmt.Errorf("parsing with cryptoinfo: %w", err)
	}

	// convert to json, so it's parsable
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	return dataflatten.Json(jsonBytes, opts...)
}
