package netskope

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFakeNsdiag installs a stand-in nsdiag shell script in dir so the exec
// path can be exercised without a Netskope install.
func writeFakeNsdiag(t *testing.T, dir, body string) {
	t.Helper()

	script := "#!/bin/sh\n" + body + "\n"
	path := filepath.Join(dir, nsdiagBinaryName())
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755)) // #nosec G306
}

func TestDefaultRunNsdiagMissingBinary(t *testing.T) {
	t.Parallel()

	_, err := defaultRunNsdiag(t.Context(), t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nsdiag not found at")
}

func TestDefaultRunNsdiag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in nsdiag is a POSIX shell script")
	}
	t.Parallel()

	dir := t.TempDir()
	writeFakeNsdiag(t, dir, `printf 'Client status:: enable.\nClient version:: 117.1.0.1234.\n'`)

	got, err := defaultRunNsdiag(t.Context(), dir)
	require.NoError(t, err)
	assert.Equal(t, "enable", got["client_status"])
	assert.Equal(t, "117.1.0.1234", got["client_version"])
}

// TestDefaultRunNsdiagSurfacesStderr covers the reason nsdiagError exists: the
// bare exit status doesn't say why nsdiag refused, but its stderr does.
func TestDefaultRunNsdiagSurfacesStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in nsdiag is a POSIX shell script")
	}
	t.Parallel()

	dir := t.TempDir()
	writeFakeNsdiag(t, dir, `printf 'nsdiag: access denied' >&2; exit 1`)

	_, err := defaultRunNsdiag(t.Context(), dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestDefaultDetectInstallPath(t *testing.T) {
	t.Parallel()

	// The host running the tests is not expected to have Netskope installed, but
	// don't fail the suite if it does.
	if path := defaultDetectInstallPath(); path != "" {
		assert.Contains(t, installPathCandidates(), path)
	}
}
