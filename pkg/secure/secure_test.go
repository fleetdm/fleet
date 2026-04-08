package secure

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkdirAll(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "test")
	require.NoError(t, os.MkdirAll(basePath, 0700))
	err := MkdirAll(basePath, 0677)
	require.Error(t, err)
	expectedErr := fmt.Sprintf(
		"Path %s already exists with mode 20000000700 instead of the expected %o", basePath, 0677^os.ModeDir)
	require.Equal(t, expectedErr, err.Error())

	err = MkdirAll(filepath.Join(basePath, "test2", "test3"), 0677)
	require.Error(t, err)
	require.Equal(t, expectedErr, err.Error())

	err = MkdirAll(filepath.Join(basePath, "test2", "test3"), 0700)
	require.NoError(t, err)
}

func TestOpenFile(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "test")
	require.NoError(t, os.MkdirAll(basePath, 0755))

	filePath := filepath.Join(basePath, "file1")
	_, err := OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0677)
	require.Error(t, err)
	expectedErr := fmt.Sprintf(
		"Path %s already exists with mode 20000000755 instead of the expected %o", basePath, 0677^os.ModeDir)
	require.Equal(t, expectedErr, err.Error())

	fd, err := OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0755)
	require.NoError(t, err)
	require.NotNil(t, fd)
	require.NoError(t, fd.Close())

	// Opening with a different perm should self-heal via chmod rather than error.
	fd, err = OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	fd.Close()
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode())

	// Re-open with the now-correct mode still works.
	fd, err = OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	fd.Close()
}
