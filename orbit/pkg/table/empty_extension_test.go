package table

import (
	"context"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyExtension(t *testing.T) {
	msg := "install orbit for more info"
	ext := EmptyExtension{name: "empty_table", msg: msg}

	assert.Equal(t, "empty_table", ext.Name())
	assert.Equal(t, []table.ColumnDefinition{table.TextColumn("message")}, ext.Columns())

	rows, err := ext.GenerateFunc(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, msg, rows[0]["message"])
}
