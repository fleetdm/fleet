//go:build darwin
// +build darwin

package find_cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// TestGenerate tests the find_cmd table generation.
func TestGenerate(t *testing.T) {
	// Test not setting required column directory.
	_, err := Generate(context.Background(), table.QueryContext{})
	require.Error(t, err)

	testDir := t.TempDir()

	// Test with an empty directory.
	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, rows[0]["path"], testDir)

	// Test with invalid type argument.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"type": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "z",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Test with invalid perm argument.
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"perm": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "foobar",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Populate the directory.
	f, err := os.Create(filepath.Join(testDir, "foo.txt"))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	err = os.Chmod(filepath.Join(testDir, "foo.txt"), os.ModePerm)
	require.NoError(t, err)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(testDir, "zoo"), os.ModePerm)
	require.NoError(t, err)
	err = os.Chmod(filepath.Join(testDir, "zoo"), os.ModePerm)
	require.NoError(t, err)

	// Test directory with a few entries.
	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, rows[0]["path"], testDir)
	require.Equal(t, rows[1]["path"], filepath.Join(testDir, "zoo"))
	require.Equal(t, rows[2]["path"], filepath.Join(testDir, "foo.txt"))

	// Test directory with a few entries and setting the perm column.
	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"perm": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "-2",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, rows[0]["path"], filepath.Join(testDir, "zoo"))
	require.Equal(t, rows[1]["path"], filepath.Join(testDir, "foo.txt"))

	// Test directory with a few entries and setting the perm and type column.
	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"perm": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "-2",
					},
				},
			},
			"type": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "d",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, rows[0]["path"], filepath.Join(testDir, "zoo"))

	// Test with not_perm argument
	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"not_perm": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "-2",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, rows[0]["path"], testDir)

	// Test with invalid not_perm argument
	_, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"not_perm": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "invalid",
					},
				},
			},
		},
	})
	require.Error(t, err)

	// Test with mindepth and maxdepth
	err = os.MkdirAll(filepath.Join(testDir, "a/b/c"), os.ModePerm)
	require.NoError(t, err)

	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"directory": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: testDir,
					},
				},
			},
			"mindepth": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "2",
					},
				},
			},
			"maxdepth": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "3",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, rows[0]["path"], filepath.Join(testDir, "a/b"))
	require.Equal(t, rows[1]["path"], filepath.Join(testDir, "a/b/c"))
}
