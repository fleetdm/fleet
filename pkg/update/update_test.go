package update

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeDirectories(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "orbit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	opt := DefaultOptions
	opt.RootDirectory = tmpDir
	updater := Updater{opt: opt}
	err = updater.initializeDirectories()
	require.NoError(t, err)
	assertDir(t, filepath.Join(tmpDir, binDir))
	assertDir(t, filepath.Join(tmpDir, binDir, osqueryDir))
	assertDir(t, filepath.Join(tmpDir, binDir, orbitDir))
}

func assertDir(t *testing.T, path string) {
	info, err := os.Stat(path)
	assert.NoError(t, err, "stat should succeed")
	assert.True(t, info.IsDir())
}

func TestMakeRepoPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		platform string
		version  string
		expected string
	}{
		{name: "osqueryd", platform: "linux", version: "4.6.0", expected: "osqueryd/linux/4.6.0/osqueryd"},
		{name: "osqueryd", platform: "windows", version: "3.3.2", expected: "osqueryd/windows/3.3.2/osqueryd.exe"},
	}

	for _, tt := range testCases {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, makeRepoPath(tt.name, tt.platform, tt.version))
		})
	}
}
