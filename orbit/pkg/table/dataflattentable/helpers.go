package dataflattentable

import (
	"github.com/kolide/launcher/pkg/dataflatten"
	"github.com/osquery/osquery-go/plugin/table"
)

// ToMap is a helper function to convert Flatten output directly for
// consumption by osquery tables.
func ToMap(rows []dataflatten.Row, query string, rowData map[string]string) []map[string]string {
	results := make([]map[string]string, len(rows))

	for i, row := range rows {
		res := make(map[string]string, len(rowData)+5)
		for k, v := range rowData {
			res[k] = v
		}

		p, k := row.ParentKey("/")

		res["fullkey"] = row.StringPath("/")
		res["parent"] = p
		res["key"] = k
		res["value"] = row.Value
		res["query"] = query

		results[i] = res
	}

	return results
}

// Columns returns the standard data flatten columns, plus whatever
// ones have been provided as additional. This is syntantic sugar for
// dataflatten based tables.
func Columns(additional ...table.ColumnDefinition) []table.ColumnDefinition {
	columns := []table.ColumnDefinition{
		table.TextColumn("fullkey"),
		table.TextColumn("parent"),
		table.TextColumn("key"),
		table.TextColumn("value"),
		table.TextColumn("query"),
	}

	return append(columns, additional...)
}
