package go_packages

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadGoBinaryWithTestExecutable(t *testing.T) {
	// The test binary itself is a Go binary, so this should succeed.
	exe, err := os.Executable()
	require.NoError(t, err)

	row := readGoBinary(exe)
	require.NotNil(t, row)
	require.NotEmpty(t, row["go_version"])
	require.Equal(t, filepath.Base(exe), row["name"])
	require.Equal(t, exe, row["installed_path"])
}

func TestReadGoBinaryNonGo(t *testing.T) {
	// /usr/bin/true is not a Go binary on any platform.
	row := readGoBinary("/usr/bin/true")
	require.Nil(t, row)
}

func TestReadGoBinaryNonExistent(t *testing.T) {
	row := readGoBinary("/nonexistent/path/binary")
	require.Nil(t, row)
}

func TestGenerateForDirsEmpty(t *testing.T) {
	results := generateForDirs(nil)
	require.Nil(t, results)
}

func TestGenerateForDirsNonExistentDir(t *testing.T) {
	results := generateForDirs([]string{"/nonexistent/home/user"})
	require.Nil(t, results)
}

func TestGenerateForDirsWithGoBinary(t *testing.T) {
	// Get the test executable — it's a real Go binary with valid build info.
	exe, err := os.Executable()
	require.NoError(t, err)

	// Create a temp directory structure: <tmpdir>/go/bin/
	tmpHome := t.TempDir()
	goBinDir := filepath.Join(tmpHome, "go", "bin")
	require.NoError(t, os.MkdirAll(goBinDir, 0o755))

	// Copy the test binary into the go/bin directory.
	destPath := filepath.Join(goBinDir, "testbinary")
	copyFile(t, exe, destPath)

	results := generateForDirs([]string{tmpHome})
	require.Len(t, results, 1)
	require.Equal(t, "testbinary", results[0]["name"])
	require.NotEmpty(t, results[0]["go_version"])
	require.Equal(t, destPath, results[0]["installed_path"])
}

// copyFile copies src to dst for testing purposes.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	require.NoError(t, err)
	defer in.Close()
	out, err := os.Create(dst)
	require.NoError(t, err)
	defer out.Close()
	_, err = io.Copy(out, in)
	require.NoError(t, err)
}
