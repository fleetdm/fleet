package filecontents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithExactPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "example.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\nworld\n"), 0o600))

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			columnPath: {
				Constraints: []table.Constraint{{
					Expression: path,
					Operator:   table.OperatorEquals,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, path, rows[0][columnPath])
	require.Equal(t, "hello\nworld\n", rows[0][columnContents])
}

func TestGenerateWithWildcard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	defer os.RemoveAll(dir)

	paths := []string{
		filepath.Join(dir, "foo.txt"),
		filepath.Join(dir, "bar.txt"),
	}

	for _, path := range paths {
		require.NoError(t, os.WriteFile(path, []byte(filepath.Base(path)+"\n"+filepath.Base(path)+"\n"), 0o600))
	}

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			columnPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.txt"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, len(paths))

	got := make(map[string]string, len(rows))
	for _, row := range rows {
		got[row[columnPath]] = row[columnContents]
	}

	for _, path := range paths {
		require.Contains(t, got, path)
		require.Equal(t, filepath.Base(path)+"\n"+filepath.Base(path)+"\n", got[path])
	}
}
