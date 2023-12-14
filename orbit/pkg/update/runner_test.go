package update

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
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
		Targets:       []string{"osqueryd"},
	}
	// NewRunner should not fail if targets do not exist locally.
	r, err := NewRunner(u, runnerOpts)
	require.NoError(t, err)
	execPath, err := u.ExecutableLocalPath("osqueryd")
	require.NoError(t, err)
	require.NoFileExists(t, execPath)

	// r.UpdateAction should download osqueryd.
	didUpdate, err := r.UpdateAction()
	require.NoError(t, err)
	require.True(t, didUpdate)
	require.FileExists(t, execPath)

	// Create another Runner but with the target already existing.
	r2, err := NewRunner(u, runnerOpts)
	require.NoError(t, err)

	didUpdate, err = r2.UpdateAction()
	require.NoError(t, err)
	require.False(t, didUpdate)
}

func TestRandomizeDuration(t *testing.T) {
	rand, err := randomizeDuration(time.Minute, 10*time.Minute)
	require.NoError(t, err)
	assert.True(t, rand >= time.Minute)
	assert.True(t, rand < 10*time.Minute)
}
