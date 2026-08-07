package ai_tools

import (
	"context"
	"fmt"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns returns the column definitions for the unified ai_tools table.
//
// This is an exported wrapper over the vendored (unexported) columnDefs so the
// table can be registered from orbit's extension via table.NewPlugin.
func Columns() []table.ColumnDefinition {
	return columnDefs()
}

// Generate produces rows for the ai_tools table for a given query.
//
// This is an exported wrapper over the vendored (unexported) generate so the
// table can be registered from orbit's extension via table.NewPlugin.
//
// It recovers from any panic in the collectors: this table runs in-process in
// the root/SYSTEM orbit daemon, parsing untrusted plist/JSON/YAML/TOML/zip from
// every user home, and a single malformed file must not crash the daemon's
// whole custom-table surface. A recovered panic is turned into a query error.
func Generate(ctx context.Context, qc table.QueryContext) (rows []map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			rows, err = nil, fmt.Errorf("ai_tools: recovered from panic: %v", r)
		}
	}()
	return generate(ctx, qc)
}
