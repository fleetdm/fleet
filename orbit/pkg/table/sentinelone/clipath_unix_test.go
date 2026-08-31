//go:build darwin || linux

package sentinelone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstExistingPath(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "sentinelctl")
	require.NoError(t, os.WriteFile(present, []byte("stub"), 0o644))
	missing := filepath.Join(dir, "missing", "sentinelctl")

	assert.Equal(t, present, firstExistingPath([]string{missing, present}))
	assert.Equal(t, present, firstExistingPath([]string{present, missing}))
	assert.Empty(t, firstExistingPath([]string{missing}))
	assert.Empty(t, firstExistingPath(nil))
}

// TestResolveCLIPathCandidates asserts the candidate list is ordered with the
// installer-created symlink first, since that is the only path guaranteed
// stable across agent versions.
func TestResolveCLIPathCandidates(t *testing.T) {
	require.NotEmpty(t, cliCandidatePaths)
	assert.Equal(t, "/usr/local/bin/sentinelctl", cliCandidatePaths[0])
	for _, p := range cliCandidatePaths {
		assert.True(t, filepath.IsAbs(p), "candidate %q must be absolute", p)
	}
}

// TestRunSentinelctlNotInstalled covers the real runner, not a mock: with no
// sentinelctl on disk it must report the binary as missing rather than
// resolving one through $PATH.
func TestRunSentinelctlNotInstalled(t *testing.T) {
	original := cliCandidatePaths
	cliCandidatePaths = []string{filepath.Join(t.TempDir(), "sentinelctl")}
	t.Cleanup(func() { cliCandidatePaths = original })

	out, err := runSentinelctl(t.Context(), "status")
	require.ErrorIs(t, err, errCLINotFound)
	assert.Empty(t, out)
}
