//go:build windows

package cisaudit

import (
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestGenerateItemNotPresent(t *testing.T) {
	ctx := context.Background()

	queryContext := table.QueryContext{
		Constraints: make(map[string]table.ConstraintList),
	}
	result, err := Generate(ctx, queryContext)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Empty(t, result[0]["item"])
	assert.Empty(t, result[0]["value"])
}

func TestGenerateItemConstrainIsPresentAndResponseMaintainsValue(t *testing.T) {
	ctx := context.Background()

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"item": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "value",
					},
				},
			},
		},
	}
	result, err := Generate(ctx, queryContext)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "value", result[0]["item"])
}

func TestGenerateItemInvalidInput(t *testing.T) {
	ctx := context.Background()

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"item": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "9.9.9.9.9.9",
					},
				},
			},
		},
	}
	result, err := Generate(ctx, queryContext)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Empty(t, result[0]["value"])
}

// TestGenerateItemValid queries a real CIS audit item via secedit.
// The CI runner (windows-latest) runs elevated so secedit succeeds.
func TestGenerateItemValid(t *testing.T) {
	ctx := context.Background()

	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"item": {
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "1.2.1",
					},
				},
			},
		},
	}
	result, err := Generate(ctx, queryContext)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "1.2.1", result[0]["item"])
	assert.NotEmpty(t, result[0]["value"])
}
