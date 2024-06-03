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
			"client": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "test-sys-client-1",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, rows[0]["service"], "test-sys-service-1")

	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"source": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "user",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 62)
	for _, row := range rows {
		require.NotEqual(t, row["source"], "system")
	}
}
