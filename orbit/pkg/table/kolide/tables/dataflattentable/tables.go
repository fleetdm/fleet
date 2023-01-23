package dataflattentable

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

type DataSourceType int

const (
	PlistType DataSourceType = iota + 1
	JsonType
	ExecType
	XmlType
	IniType
	KeyValueType
)

type Table struct {
	client    *osquery.ExtensionManagerClient
	logger    log.Logger
	tableName string

	flattenFileFunc  func(string, ...dataflatten.FlattenOpts) ([]dataflatten.Row, error)
	flattenBytesFunc func([]byte, ...dataflatten.FlattenOpts) ([]dataflatten.Row, error)

	execArgs []string
	binDirs  []string

	keyValueSeparator string
}

// AllTablePlugins is a helper to return all the expected flattening tables.
func AllTablePlugins(client *osquery.ExtensionManagerClient, logger log.Logger) []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		TablePlugin(client, logger, JsonType),
		TablePlugin(client, logger, XmlType),
		TablePlugin(client, logger, IniType),
		TablePlugin(client, logger, PlistType),
	}
}

func TablePlugin(client *osquery.ExtensionManagerClient, logger log.Logger, dataSourceType DataSourceType) osquery.OsqueryPlugin {
	columns := Columns(table.TextColumn("path"))

	t := &Table{
		client: client,
		logger: logger,
	}

	switch dataSourceType {
	case PlistType:
		t.flattenFileFunc = dataflatten.PlistFile
		t.tableName = "kolide_plist"
	case JsonType:
		t.flattenFileFunc = dataflatten.JsonFile
		t.tableName = "kolide_json"
	case XmlType:
		t.flattenFileFunc = dataflatten.XmlFile
		t.tableName = "kolide_xml"
	case IniType:
		t.flattenFileFunc = dataflatten.IniFile
		t.tableName = "kolide_ini"
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
					level.Info(t.logger).Log(
						"msg", "failed to get data for path",
						"path", filePath,
						"err", err,
					)
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
		level.Info(t.logger).Log("msg", "failure parsing file", "file", filePath)
		return nil, fmt.Errorf("parsing data: %w", err)
	}

	rowData := map[string]string{
		"path": filePath,
	}

	return ToMap(data, dataQuery, rowData), nil
}
