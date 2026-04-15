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

func TestWriteXAR_NativeXarCanRead(t *testing.T) {
	// Verify that macOS native xar can list and extract our Go-generated XAR.
	if _, err := exec.LookPath("xar"); err != nil {
		t.Skip("xar not found, skipping native compat test")
	}

	tmpDir := t.TempDir()
	flatDir := filepath.Join(tmpDir, "flat")
	basePkg := filepath.Join(flatDir, "base.pkg")
	require.NoError(t, os.MkdirAll(basePkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(flatDir, "Distribution"), []byte("<installer/>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "PackageInfo"), []byte("<pkg-info/>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "Bom"), []byte("bom-data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "Payload"), []byte("payload-data"), 0o644))

	outputPath := filepath.Join(tmpDir, "test.pkg")
	require.NoError(t, writeXAR(flatDir, outputPath))

	// List with xar -tf.
	out, err := exec.Command("xar", "-tf", outputPath).CombinedOutput()
	require.NoError(t, err, "xar -tf failed: %s", out)

	listing := strings.TrimSpace(string(out))
	lines := strings.Split(listing, "\n")

	// Should contain all files.
	assert.Contains(t, lines, "Distribution")
	assert.Contains(t, lines, "base.pkg")

	// Extract and verify contents.
	extractDir := filepath.Join(tmpDir, "extracted")
	require.NoError(t, os.MkdirAll(extractDir, 0o755))
	out, err = exec.Command("xar", "-xf", outputPath, "-C", extractDir).CombinedOutput()
	require.NoError(t, err, "xar -xf failed: %s", out)

	// Verify extracted file contents.
	distContent, err := os.ReadFile(filepath.Join(extractDir, "Distribution"))
	require.NoError(t, err)
	assert.Equal(t, "<installer/>", string(distContent))

	pkgInfoContent, err := os.ReadFile(filepath.Join(extractDir, "base.pkg", "PackageInfo"))
	require.NoError(t, err)
	assert.Equal(t, "<pkg-info/>", string(pkgInfoContent))

	payloadContent, err := os.ReadFile(filepath.Join(extractDir, "base.pkg", "Payload"))
	require.NoError(t, err)
	assert.Equal(t, "payload-data", string(payloadContent))
}
