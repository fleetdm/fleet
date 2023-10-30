//go:build windows
// +build windows

package wmitable

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/wmi"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const allowedCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("namespace"),
		table.TextColumn("class"),
		table.TextColumn("properties"),
		table.TextColumn("whereclause"),
	)

	t := &Table{
		logger: level.NewFilter(logger),
	}

	return table.NewPlugin("wmi", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	classes := tablehelpers.GetConstraints(queryContext, "class", tablehelpers.WithAllowedCharacters(allowedCharacters))
	if len(classes) == 0 {
		return nil, errors.New("The kolide_wmi table requires a wmi class")
	}

	properties := tablehelpers.GetConstraints(queryContext, "properties", tablehelpers.WithAllowedCharacters(allowedCharacters+`,`))
	if len(properties) == 0 {
		return nil, errors.New("The kolide_wmi table requires wmi properties")
	}

	// Get the list of namespaces to query. If not specified, that's
	// okay too, default to ""
	namespaces := tablehelpers.GetConstraints(queryContext, "namespace",
		tablehelpers.WithDefaults(""),
		tablehelpers.WithAllowedCharacters(allowedCharacters+`\`),
	)

	// Any whereclauses? These are not required
	whereClauses := tablehelpers.GetConstraints(queryContext, "whereclause",
		tablehelpers.WithDefaults(""),
		tablehelpers.WithAllowedCharacters(allowedCharacters+`:\= '".`),
	)

	flattenQueries := tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*"))

	for _, class := range classes {
		for _, rawProperties := range properties {
			properties := strings.Split(rawProperties, ",")
			if len(properties) == 0 {
				continue
			}
			for _, ns := range namespaces {
				// The namespace argument uses a bare
				// backslash, not a doubled one. But,
				// it's common to double backslashes
				// to escape them through quoting
				// blocks. We can collapse them it
				// down here, and create a small ux
				// improvement.
				ns = strings.ReplaceAll(ns, `\\`, `\`)

				for _, whereClause := range whereClauses {
					// Set a timeout in case wmi hangs
					ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
					defer cancel()

					wmiResults, err := wmi.Query(ctx, t.logger, class, properties, wmi.ConnectUseMaxWait(), wmi.ConnectNamespace(ns), wmi.WithWhere(whereClause))
					if err != nil {
						level.Info(t.logger).Log(
							"msg", "wmi query failure",
							"err", err,
							"class", class,
							"properties", rawProperties,
							"namespace", ns,
							"whereClause", whereClause,
						)
						continue
					}

					for _, dataQuery := range flattenQueries {
						results = append(results, t.flattenRowsFromWmi(dataQuery, wmiResults, class, rawProperties, ns, whereClause)...)
					}
				}
			}
		}
	}

	return results, nil
}

func (t *Table) flattenRowsFromWmi(dataQuery string, wmiResults []map[string]interface{}, wmiClass, wmiProperties, wmiNamespace, whereClause string) []map[string]string {
	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	// wmi.Query returns []map[string]interface{}, but dataflatten
	// wants it as []interface{}. So let's whomp it.
	resultsCasted := make([]interface{}, len(wmiResults))
	for i, r := range wmiResults {
		resultsCasted[i] = r
	}

	flatData, err := dataflatten.Flatten(resultsCasted, flattenOpts...)
	if err != nil {
		level.Info(t.logger).Log("msg", "failure flattening output", "err", err)
		return nil
	}

	rowData := map[string]string{
		"class":       wmiClass,
		"properties":  wmiProperties,
		"namespace":   wmiNamespace,
		"whereclause": whereClause,
	}

	return dataflattentable.ToMap(flatData, dataQuery, rowData)
}
