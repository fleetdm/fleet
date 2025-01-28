// based on github.com/kolide/launcher/pkg/osquery/tables
package dataflattentable

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

type DataSourceType int

const (
	PlistType DataSourceType = iota + 1
	JsonType
	JsonlType
	ExecType
	XmlType
	IniType
	KeyValueType
)

type Table struct {
	logger    zerolog.Logger
	tableName string

	flattenFileFunc  func(string, ...dataflatten.FlattenOpts) ([]dataflatten.Row, error)
	flattenBytesFunc func([]byte, ...dataflatten.FlattenOpts) ([]dataflatten.Row, error)

	execArgs []string
	binDirs  []string

	keyValueSeparator string
}

// AllTablePlugins is a helper to return all the expected flattening tables.
func AllTablePlugins(logger zerolog.Logger) []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		TablePlugin(logger, JsonType),
		TablePlugin(logger, XmlType),
		TablePlugin(logger, IniType),
		TablePlugin(logger, PlistType),
		TablePlugin(logger, JsonlType),
	}
}

func TablePlugin(logger zerolog.Logger, dataSourceType DataSourceType) osquery.OsqueryPlugin {
	columns := Columns(table.TextColumn("path"))

	t := &Table{
		logger: logger,
	}

	switch dataSourceType {
	case PlistType:
		t.flattenFileFunc = dataflatten.PlistFile
		t.tableName = "parse_plist"
	case JsonType:
		t.flattenFileFunc = dataflatten.JsonFile
		t.tableName = "parse_json"
	case JsonlType:
		t.flattenFileFunc = dataflatten.JsonlFile
		t.tableName = "parse_jsonl"
	case XmlType:
		t.flattenFileFunc = dataflatten.XmlFile
		t.tableName = "parse_xml"
	case IniType:
		t.flattenFileFunc = dataflatten.IniFile
		t.tableName = "parse_ini"
	default:
		panic("Unknown data source type")
	}

	return table.NewPlugin(t.tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedPaths := tablehelpers.GetConstraints(queryContext, "path")
	if len(requestedPaths) == 0 {
		return results, fmt.Errorf("The %s table requires that you specify a single constraint for path", t.tableName)
	}

	for _, requestedPath := range requestedPaths {

		// We take globs in via the sql %, but glob needs *. So convert.
		filePaths, err := filepath.Glob(strings.ReplaceAll(requestedPath, `%`, `*`))
		if err != nil {
			return results, fmt.Errorf("bad glob: %w", err)
		}

		for _, filePath := range filePaths {
			for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
				subresults, err := t.generatePath(filePath, dataQuery)
				if err != nil {
					t.logger.Info().Err(err).Str("path", filePath).Msg("failed to get data for path")
					continue
				}

				results = append(results, subresults...)
			}
		}
	}
	return results, nil
}

func (t *Table) generatePath(filePath string, dataQuery string) ([]map[string]string, error) {
	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithNestedPlist(),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	data, err := t.flattenFileFunc(filePath, flattenOpts...)
	if err != nil {
		t.logger.Info().Err(err).Str("file", filePath).Msg("failure parsing file")
		return nil, fmt.Errorf("parsing data: %w", err)
	}

	rowData := map[string]string{
		"path": filePath,
	}

	return ToMap(data, dataQuery, rowData), nil
}
