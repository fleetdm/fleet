//go:build darwin

package executable_hashes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithExactPath(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "example.bin")
	execPath := path
	content := []byte("test file content for hashing")
	require.NoError(t, os.WriteFile(path, content, 0o600))

	h := sha256.New()
	h.Write(content)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: path,
					Operator:   table.OperatorEquals,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, path, rows[0][colPath])
	require.Equal(t, execPath, rows[0][colExecPath])
	require.Equal(t, expectedHash, rows[0][colExecHash])
}

func TestGenerateWithWildcard(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	testFiles := map[string][]byte{
		"foo.bin": []byte("content of foo"),
		"bar.bin": []byte("content of bar"),
		"baz.bin": []byte("content of baz"),
	}

	expectedHashByBundlePath := make(map[string]string)

	for filename, content := range testFiles {
		path := filepath.Join(dir, filename)
		require.NoError(t, os.WriteFile(path, content, 0o600))

		h := sha256.New()
		h.Write(content)
		expectedHashByBundlePath[path] = hex.EncodeToString(h.Sum(nil))
	}

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.bin"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, len(testFiles))

	got := make(map[string]fileInfo, len(rows))
	for _, row := range rows {
		got[row[colPath]] = fileInfo{
			Path:       row[colPath],
			ExecPath:   row[colExecPath],
			ExecSha256: row[colExecHash],
		}
	}

	for path, expectedHash := range expectedHashByBundlePath {
		require.Contains(t, got, path)
		info := got[path]
		require.Equal(t, path, info.Path)
		require.Equal(t, path, info.ExecPath)
		require.Equal(t, expectedHash, info.ExecSha256)
	}
}
