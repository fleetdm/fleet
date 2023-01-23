package cryptoinfotable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/cryptoinfo"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("passphrase"),
		table.TextColumn("path"),
	)

	t := &Table{
		logger: logger,
	}

	return table.NewPlugin("kolide_cryptoinfo", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedPaths := tablehelpers.GetConstraints(queryContext, "path")
	if len(requestedPaths) == 0 {
		return results, errors.New("The kolide_cryptoinfo table requires that you specify an equals constraint for path")
	}

	for _, requestedPath := range requestedPaths {

		// We take globs in via the sql %, but glob needs *. So convert.
		filePaths, err := filepath.Glob(strings.ReplaceAll(requestedPath, `%`, `*`))
		if err != nil {
			level.Info(t.logger).Log("msg", "bad file glob", "err", err)
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
						level.Info(t.logger).Log(
							"msg", "failed to get data for path",
							"path", filePath,
							"err", err,
						)
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
