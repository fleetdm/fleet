//go:build darwin
// +build darwin

package find_cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	err = os.Chmod(filepath.Join(testDir, "foo.txt"), os.ModePerm) // nolint:gosec // G302
	require.NoError(t, err)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(testDir, "zoo"), os.ModePerm)
	require.NoError(t, err)
	err = os.Chmod(filepath.Join(testDir, "zoo"), os.ModePerm) // nolint:gosec // G302
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
}

// generateWithGuard fails the test instead of hanging the suite if Generate never returns.
func generateWithGuard(t *testing.T, directory string) []map[string]string {
	t.Helper()
	type result struct {
		rows []map[string]string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		rows, err := Generate(t.Context(), table.QueryContext{
			Constraints: map[string]table.ConstraintList{
				"directory": {
					Affinity:    table.ColumnTypeText,
					Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: directory}},
				},
			},
		})
		done <- result{rows: rows, err: err}
	}()
	select {
	case r := <-done:
		require.NoError(t, r.err)
		return r.rows
	case <-time.After(10 * time.Second):
		require.FailNow(t, "Generate did not return")
		return nil
	}
}

func rowPaths(rows []map[string]string) []string {
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row["path"])
	}
	return paths
}

func TestGenerateNewlineInFilename(t *testing.T) {
	for _, name := range []string{"evil\n", "evil\nname", "\n"} {
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			testDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(testDir, name), nil, 0o600))

			rows := generateWithGuard(t, testDir)
			require.ElementsMatch(t, []string{testDir, filepath.Join(testDir, name)}, rowPaths(rows))
		})
	}
}

func TestGenerateLargeStderr(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read 0o000 directories, so find reports no errors")
	}

	// Enough unreadable directories for find's errors to overflow the stderr pipe buffer.
	testDir := t.TempDir()
	prefix := strings.Repeat("d", 150)
	var unreadable []string
	for i := range 1000 {
		dir := filepath.Join(testDir, fmt.Sprintf("%s%04d", prefix, i))
		require.NoError(t, os.Mkdir(dir, 0o000))
		unreadable = append(unreadable, dir)
	}

	rows := generateWithGuard(t, testDir)
	require.ElementsMatch(t, append([]string{testDir}, unreadable...), rowPaths(rows))
}
