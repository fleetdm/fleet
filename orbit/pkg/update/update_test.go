package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeDirectories(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.Chmod(tmpDir, constant.DefaultDirMode))

	opt := DefaultOptions
	opt.RootDirectory = tmpDir
	updater := Updater{opt: opt}
	err := updater.initializeDirectories()
	require.NoError(t, err)
	assertDir(t, filepath.Join(tmpDir, binDir))
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
		version  string
		platform string
		expected string
	}{
		{platform: "linux", name: "osqueryd", version: "4.6.0", expected: "osqueryd/linux/4.6.0/osqueryd"},
		{platform: "linux", name: "osqueryd", version: "3.3.2", expected: "osqueryd/linux/3.3.2/osqueryd"},
		{platform: "macos", name: "osqueryd", version: "4.6.0", expected: "osqueryd/macos/4.6.0/osqueryd"},
		{platform: "macos", name: "osqueryd", version: "3.3.2", expected: "osqueryd/macos/3.3.2/osqueryd"},
		{platform: "windows", name: "osqueryd", version: "4.6.0", expected: "osqueryd/windows/4.6.0/osqueryd.exe"},
		{platform: "windows", name: "osqueryd", version: "3.3.2", expected: "osqueryd/windows/3.3.2/osqueryd.exe"},
	}

	for _, tt := range testCases {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			u := Updater{opt: Options{Platform: tt.platform}}
			assert.Equal(t, tt.expected, u.RepoPath(tt.name, tt.version))
		})
	}
}
