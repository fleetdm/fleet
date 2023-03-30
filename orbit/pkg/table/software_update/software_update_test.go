//go:build darwin
// +build darwin

package software_update

import (
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var tbl table.QueryContext

	table, err := Generate(ctx, tbl)
	require.Nil(t, err)

	if table[0]["software_update_required"] != "0" && table[0]["software_update_required"] != "1" {
		t.Fatalf(`software_update_required expected 0 or 1. got %s`, table[0]["software_update_required"])
	}
}

func TestIsNewSoftwareAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newSoftwareAvailable, err := isNewSoftwareAvailable(ctx)
	if newSoftwareAvailable != "0" && newSoftwareAvailable != "1" {
		t.Fatalf(`newSoftwareAvailable expected 0 or 1. got %s`, newSoftwareAvailable)
	}
	require.Nil(t, err)
}

func TestColumns(t *testing.T) {
	col := Columns()
	require.Equal(t, []table.ColumnDefinition{{Name: "software_update_required", Type: "INTEGER"}}, col)
}
