package packaging

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBOM_NativeLsbomCanRead(t *testing.T) {
	// Verify that macOS native lsbom can read our Go-generated BOM.
	if _, err := exec.LookPath("lsbom"); err != nil {
		t.Skip("lsbom not found, skipping native compat test")
	}

	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "opt", "orbit"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "Library", "LaunchDaemons"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, "opt", "orbit", "secret.txt"),
		[]byte("fleet-secret"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, "Library", "LaunchDaemons", "com.fleetdm.orbit.plist"),
		[]byte("<plist>test</plist>"),
		0o644,
	))

	bomPath := filepath.Join(tmpDir, "test.bom")
	require.NoError(t, writeBOM(rootDir, bomPath, 0, 80))

	// Run native lsbom.
	out, err := exec.Command("lsbom", bomPath).CombinedOutput()
	require.NoError(t, err, "lsbom failed: %s", out)

	listing := string(out)

	// Should contain all our paths.
	assert.Contains(t, listing, ".")
	assert.Contains(t, listing, "Library")
	assert.Contains(t, listing, "LaunchDaemons")
	assert.Contains(t, listing, "com.fleetdm.orbit.plist")
	assert.Contains(t, listing, "opt")
	assert.Contains(t, listing, "orbit")
	assert.Contains(t, listing, "secret.txt")

	// Verify uid/gid are 0/80.
	for line := range strings.SplitSeq(listing, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		assert.Contains(t, line, "0/80", "all entries should have uid=0 gid=80: %s", line)
	}
}
