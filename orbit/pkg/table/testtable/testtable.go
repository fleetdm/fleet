//go:build darwin
// +build darwin

package testtable

import (
	"context"
	"errors"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("input_col"), // required
		table.TextColumn("output_col"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	inputCol := ""
	if constraints, ok := queryContext.Constraints["input_col"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				inputCol = constraint.Expression
			}
		}
	}
	if inputCol == "" {
		return nil, errors.New("missing input_col")
	}

	return []map[string]string{{
		"input_col":  inputCol,
		"output_col": inputCol + "_edited",
	}}, nil
}
