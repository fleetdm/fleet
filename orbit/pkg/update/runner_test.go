package update

import (
	"math/rand"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	// TODO(lucas): Do not use our TUF remote repository
	// but instead create local repository and serve with a httptest server.
	// For that, we need to move and export some functionality currently in
	// "ee/fleetctl/updates.go" (as it doesn't make sense to have such functionality
	// there and import such eefleetctl package here).
	nettest.Run(t)

	rootDir := t.TempDir()
	updateOpts := DefaultOptions
	updateOpts.RootDirectory = rootDir

	u, err := NewUpdater(updateOpts)
	require.NoError(t, err)

	err = u.UpdateMetadata()
	require.NoError(t, err)

	runnerOpts := RunnerOptions{
		CheckInterval: 1 * time.Second,
		Targets:       []string{constant.OsqueryTUFTargetName},
	}

	// NewRunner should not fail if targets do not exist locally.
	r, err := NewRunner(u, runnerOpts)
	require.NoError(t, err)

	// ExecutableLocalPath fails if the target does not exist in the expected path.
	execPath, err := u.ExecutableLocalPath(constant.OsqueryTUFTargetName)
	require.Error(t, err)
	require.NoFileExists(t, execPath)

	// r.UpdateAction should download osqueryd.
	didUpdate, err := r.UpdateAction()
	require.NoError(t, err)
	require.True(t, didUpdate)

	// ExecutableLocalPath should now succeed.
	execPath, err = u.ExecutableLocalPath(constant.OsqueryTUFTargetName)
	require.NoError(t, err)
	require.FileExists(t, execPath)

	// Create another Runner but with the target already existing.
	r2, err := NewRunner(u, runnerOpts)
	require.NoError(t, err)

	didUpdate, err = r2.UpdateAction()
	require.NoError(t, err)
	require.False(t, didUpdate)
}

func TestRandomizeDuration(t *testing.T) {
	rand, err := randomizeDuration(10 * time.Minute)
	require.NoError(t, err)
	assert.True(t, rand >= 0)
	assert.True(t, rand < 10*time.Minute)
}

func TestGetVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}
	t.Parallel()
	testCases := map[string]struct {
		cmd     string
		version string
	}{
		"4.5.6": {
			cmd:     "#!/bin/bash\n/bin/echo orbit 4.5.6",
			version: "4.5.6",
		},
		"42.0.0": {
			cmd:     "#!/bin/bash\n/bin/echo fleet-desktop 42.0.0",
			version: "42.0.0",
		},
		"5.10.2-26-gc396d07b4-dirty": {
			cmd:     "#!/bin/bash\n/bin/echo osquery version 5.10.2-26-gc396d07b4-dirty",
			version: "5.10.2-26-gc396d07b4-dirty",
		},
		"bad output": {
			cmd:     "#!/bin/bash\n/bin/echo osquery version is weird",
			version: "",
		},
		"bad cmd": {
			cmd:     "bozo+bozo+bozo",
			version: "",
		},
	}
	for name, tc := range testCases {
		tc := tc // capture range variable, needed for parallel tests
		t.Run(
			name, func(t *testing.T) {
				t.Parallel()
				// create a temp executable file
				dir := t.TempDir()
				file, err := os.CreateTemp(dir, "binary")
				require.NoError(t, err)
				_, err = file.WriteString(tc.cmd)
				require.NoError(t, err)
				err = file.Chmod(0o755)
				require.NoError(t, err)
				_ = file.Close()

				// "text file busy" is a Go issue when executing file just written: https://github.com/golang/go/issues/22315
				var version string
				retries := 0
				for {
					version, err = GetVersion(file.Name())
					if err != nil {
						t.Log(err)
						if strings.Contains(err.Error(), "text file busy") {
							if retries > 5 {
								t.Fatal("too many retries due to 'text file busy' error: https://github.com/golang/go/issues/22315")
							}
							// adding some randomization so that parallel tests get out of sync if needed
							time.Sleep((500 + time.Duration(rand.Intn(100))) * time.Millisecond) //nolint:gosec
							retries++
						} else {
							break
						}
					} else {
						break
					}
				}
				assert.Equal(t, tc.version, version)
			},
		)
	}
}

func TestCompareVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}
	t.Parallel()
	testCases := map[string]struct {
		cmd        string
		oldVersion string
		expected   *int
	}{
		"downgrade": {
			cmd:        "#!/bin/bash\n/bin/echo orbit 4.9",
			oldVersion: "4.10",
			expected:   ptr.Int(1),
		},
		"same": {
			cmd:        "#!/bin/bash\n/bin/echo osquery version 5.10.2-26-gc396d07b4-dirty",
			oldVersion: "5.10.2-26-gc396d07b4-dirty",
			expected:   ptr.Int(0),
		},
		"same 2": {
			cmd:        "#!/bin/bash\n/bin/echo osquery version 5.10",
			oldVersion: "5.10.0",
			expected:   ptr.Int(0),
		},
		"upgrade": {
			cmd:        "#!/bin/bash\n/bin/echo osquery version 5.10.10",
			oldVersion: "5.10.9",
			expected:   ptr.Int(-1),
		},
		"invalid new version": {
			cmd:        "#!/bin/bash\n/bin/echo osquery version invalid",
			oldVersion: "5.10.9",
			expected:   nil,
		},
		"invalid old version": {
			cmd:        "#!/bin/bash\n/bin/echo orbit 1",
			oldVersion: "",
			expected:   nil,
		},
		"invalid old version 2": {
			cmd:        "#!/bin/bash\n/bin/echo orbit 1",
			oldVersion: "1.01", // invalid, needs to be 1.1
			expected:   nil,
		},
	}
	for name, tc := range testCases {
		tc := tc // capture range variable, needed for parallel tests
		t.Run(
			name, func(t *testing.T) {
				t.Parallel()
				// create a temp executable file
				dir := t.TempDir()
				file, err := os.CreateTemp(dir, "binary")
				require.NoError(t, err)
				_, err = file.WriteString(tc.cmd)
				require.NoError(t, err)
				err = file.Chmod(0o755)
				require.NoError(t, err)
				_ = file.Close()

				// "text file busy" is a Go issue when executing file just written: https://github.com/golang/go/issues/22315
				var result *int
				retries := 0
				for {
					result, err = compareVersion(file.Name(), tc.oldVersion, "target")
					if err != nil {
						t.Log(err)
						if strings.Contains(err.Error(), "text file busy") {
							if retries > 5 {
								t.Fatal("too many retries due to 'text file busy' error: https://github.com/golang/go/issues/22315")
							}
							// adding some randomization so that parallel tests get out of sync if needed
							time.Sleep((500 + time.Duration(rand.Intn(100))) * time.Millisecond) //nolint:gosec
							retries++
						} else {
							break
						}
					} else {
						break
					}
				}
				assert.Equal(t, tc.expected, result)
			},
		)
	}
}
