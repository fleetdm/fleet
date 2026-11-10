package fleetctl

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestIsRenamedOrbit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("orbit renames its process only on Linux")
	}
	pid := os.Getpid()
	exe, err := os.Executable()
	require.NoError(t, err)
	exeDir := filepath.Dir(exe)
	require.False(t, isRenamedOrbit(pid, exeDir), "not renamed")

	// Rename this test process the way orbit renames itself.
	orig, err := os.ReadFile("/proc/self/comm")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("/proc/self/comm", []byte(constant.AgentDisplayName), 0))
	t.Cleanup(func() { _ = os.WriteFile("/proc/self/comm", bytes.TrimSpace(orig), 0) })

	require.True(t, isRenamedOrbit(pid, exeDir))
	require.False(t, isRenamedOrbit(pid, t.TempDir()), "another app named Fleet")
}
