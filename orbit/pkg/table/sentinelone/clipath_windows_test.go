//go:build windows

package sentinelone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want []int
	}{
		{in: "Sentinel Agent 25.1.4.434", want: []int{25, 1, 4, 434}},
		{in: "Sentinel Agent 23.2.4.7", want: []int{23, 2, 4, 7}},
		{in: "Sentinel Agent 25.1", want: []int{25, 1}},
		{in: "Sentinel Agent 25.1.4.434-beta", want: []int{25, 1, 4, 434}},
		{in: "Sentinel Agent", want: nil},
		{in: "Sentinel Agent 25", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, parseDirVersion(tt.in))
		})
	}
}

// TestResolveCLIPathHighestVersion covers a host left with more than one agent
// directory after an upgrade: the newest install must win, and a directory
// without SentinelCtl.exe must be ignored.
func TestResolveCLIPathHighestVersion(t *testing.T) {
	root := t.TempDir()
	original := cliInstallRoots
	cliInstallRoots = []string{root}
	t.Cleanup(func() { cliInstallRoots = original })

	withCtl := []string{"Sentinel Agent 23.2.4.7", "Sentinel Agent 25.1.4.434", "Sentinel Agent"}
	for _, dir := range withCtl {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, dir, ctlBinary), []byte("stub"), 0o644))
	}
	// A leftover directory with no binary, and an unrelated directory.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Sentinel Agent 26.0.0.1"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Logs"), 0o755))

	assert.Equal(t, filepath.Join(root, "Sentinel Agent 25.1.4.434", ctlBinary), resolveCLIPath())
}

func TestResolveCLIPathNotInstalled(t *testing.T) {
	original := cliInstallRoots
	cliInstallRoots = []string{filepath.Join(t.TempDir(), "missing")}
	t.Cleanup(func() { cliInstallRoots = original })

	assert.Empty(t, resolveCLIPath())
}

// TestResolveCLIPathInstallRoot covers the older installer layout that dropped
// the binary directly in the install root.
func TestResolveCLIPathInstallRoot(t *testing.T) {
	root := t.TempDir()
	original := cliInstallRoots
	cliInstallRoots = []string{root}
	t.Cleanup(func() { cliInstallRoots = original })

	require.NoError(t, os.WriteFile(filepath.Join(root, ctlBinary), []byte("stub"), 0o644))

	assert.Equal(t, filepath.Join(root, ctlBinary), resolveCLIPath())
}
