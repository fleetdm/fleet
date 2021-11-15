package file_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	originalPath := filepath.Join(tmp, "original")
	dstPath := filepath.Join(tmp, "copy")
	expectedContents := []byte("foo")
	expectedMode := fs.FileMode(0644)
	require.NoError(t, os.WriteFile(originalPath, expectedContents, os.ModePerm))
	require.NoError(t, os.WriteFile(dstPath, []byte("this should be overwritten"), expectedMode))

	// Test
	require.NoError(t, file.Copy(originalPath, dstPath, expectedMode))

	contents, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	contents, err = os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedMode, info.Mode())

	// Copy of nonexistent path fails
	require.Error(t, file.Copy(filepath.Join(tmp, "notexist"), dstPath, os.ModePerm))

	// Copy to nonexistent directory
	require.Error(t, file.Copy(originalPath, filepath.Join("tmp", "notexist", "foo"), os.ModePerm))
}

func TestCopyWithPerms(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	originalPath := filepath.Join(tmp, "original")
	dstPath := filepath.Join(tmp, "copy")
	expectedContents := []byte("foo")
	expectedMode := fs.FileMode(0755)
	require.NoError(t, os.WriteFile(originalPath, expectedContents, expectedMode))

	// Test
	require.NoError(t, file.CopyWithPerms(originalPath, dstPath))

	contents, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedMode, info.Mode())
}

func TestExists(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	path := filepath.Join(tmp, "file")
	require.NoError(t, os.WriteFile(path, []byte(""), os.ModePerm))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "dir", "nested"), os.ModePerm))

	// Test
	exists, err := file.Exists(path)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = file.Exists(filepath.Join(tmp, "notexist"))
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = file.Exists(filepath.Join(tmp, "dir"))
	require.NoError(t, err)
	assert.False(t, exists)
}
