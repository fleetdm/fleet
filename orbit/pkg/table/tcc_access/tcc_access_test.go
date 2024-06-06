//go:build darwin
// +build darwin

package tcc_access

import (
	"context"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

// TestGenerate tests the tcc_access table generation.
func TestGenerate(t *testing.T) {
	testContext = true
	tccPathPrefix = "./testdata"
	tccPathSuffix = "/test-TCC.db"

	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)

	require.Len(t, rows, 93)

	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"uid": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "1",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 31)
	for _, row := range rows {
		serviceName := row["service"]
		require.Contains(t, serviceName, "u1-service")
		require.NotContains(t, serviceName, "u2-service")
		require.NotContains(t, serviceName, "sys-service")
	}
}
