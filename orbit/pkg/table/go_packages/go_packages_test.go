package go_packages

import (
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
