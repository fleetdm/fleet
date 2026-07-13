package ai_tools

import (
	"context"

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
func Generate(ctx context.Context, qc table.QueryContext) ([]map[string]string, error) {
	return generate(ctx, qc)
}
