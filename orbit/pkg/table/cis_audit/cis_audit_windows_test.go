//go:build windows
// +build windows

package cisaudit

import (
	"runtime"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestGenerateItemNotPresent(t *testing.T) {
	ctx := context.Background()

	queryContext := table.QueryContext{
		Constraints: make(map[string]table.ConstraintList),
	}
	result, err := Generate(ctx, queryContext)
	assert.Nil(t, err)
	assert.Equal(t, len(result), 1)
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
	assert.Nil(t, err)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, result[0]["item"], "value")
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
	assert.Nil(t, err)
	assert.Equal(t, len(result), 1)
	assert.Empty(t, result[0]["value"])
}

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
	_, err := Generate(ctx, queryContext)

	if runtime.GOOS == "windows" {
		assert.NotNil(t, err)
	} else {
		assert.Nil(t, err)
	}
}
