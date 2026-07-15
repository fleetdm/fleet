package packaging

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// These tests exercise the pure-Go xar writer (writeXar) by feeding its output
// to Fleet's own xar decoder in pkg/file. A xar that round-trips through the
// decoder proves the encoder produces a well-formed header, a valid zlib TOC,
// and correct heap offsets/lengths (the decoder reads members back by those
// offsets). The writer is platform-independent, so these run everywhere.

const testDistribution = `<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
	<title>Fleet osquery</title>
	<product id="com.fleetdm.orbit" version="1.2.3"/>
	<pkg-ref id="com.fleetdm.orbit.base.pkg" version="1.2.3" packageIdentifier="com.fleetdm.orbit"/>
</installer-gui-script>
`

const testPackageInfo = `<pkg-info format-version="2" identifier="com.fleetdm.orbit" version="9.9.9" install-location="/" auth="root"/>
`

// writeXarTree writes the given name->contents map (names may contain "/" to
// create nested directories) into a fresh temp dir and returns its path.
func writeXarTree(t *testing.T, files map[string][]byte) string {
	t.Helper()
	root := t.TempDir()
	for name, data := range files {
		p := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, data, 0o644))
	}
	return root
}

// buildXar runs writeXar over root and returns the archive bytes.
func buildXar(t *testing.T, root string) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.pkg")
	require.NoError(t, writeXar(root, out))
	b, err := os.ReadFile(out)
	require.NoError(t, err)
	return b
}

func extractXARMetadata(t *testing.T, xarBytes []byte) *file.InstallerMetadata {
	t.Helper()
	tfr, err := fleet.NewTempFileReader(bytes.NewReader(xarBytes), t.TempDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tfr.Close() })
	meta, err := file.ExtractXARMetadata(tfr)
	require.NoError(t, err)
	return meta
}

// TestWriteXarReadableByDecoder builds a distribution-style .pkg tree and
// verifies the pure-Go writer's output is a valid xar: parseable header + TOC,
// a discoverable Distribution member, and metadata read back correctly from the
// heap.
func TestWriteXarReadableByDecoder(t *testing.T) {
	root := writeXarTree(t, map[string][]byte{
		"Distribution":         []byte(testDistribution),
		"base.pkg/PackageInfo": []byte(testPackageInfo),
		"base.pkg/Payload":     bytes.Repeat([]byte("orbit-payload-bytes\n"), 1000),
	})
	xarBytes := buildXar(t, root)

	// Valid, unsigned xar: exercises magic-byte check, SHA-1 hash-type mapping,
	// zlib TOC decompression, and TOC XML parsing.
	require.ErrorIs(t, file.CheckPKGSignature(bytes.NewReader(xarBytes)), file.ErrNotSigned)

	// The TOC lists the top-level Distribution file.
	hasDist, err := file.XARHasDistribution(bytes.NewReader(xarBytes))
	require.NoError(t, err)
	require.True(t, hasDist)

	// Reading the Distribution member back (via its <offset>/<length> within the
	// heap that begins with the 20-byte TOC checksum) yields exactly the bytes we
	// wrote; if any offset/length were off the XML parse would fail.
	meta := extractXARMetadata(t, xarBytes)
	require.Equal(t, "Fleet osquery", meta.Name)
	require.Equal(t, "1.2.3", meta.Version)
	require.Equal(t, "com.fleetdm.orbit", meta.BundleIdentifier)
	require.Contains(t, meta.PackageIDs, "com.fleetdm.orbit")
}

// TestWriteXarPackageInfoFallback verifies a component-style .pkg (top-level
// PackageInfo, no Distribution) also round-trips: the decoder falls back to
// PackageInfo, which again requires the writer's heap offsets to be correct.
func TestWriteXarPackageInfoFallback(t *testing.T) {
	root := writeXarTree(t, map[string][]byte{
		"PackageInfo": []byte(testPackageInfo),
	})
	xarBytes := buildXar(t, root)

	hasDist, err := file.XARHasDistribution(bytes.NewReader(xarBytes))
	require.NoError(t, err)
	require.False(t, hasDist)

	meta := extractXARMetadata(t, xarBytes)
	require.Equal(t, "9.9.9", meta.Version)
	require.Equal(t, "com.fleetdm.orbit", meta.BundleIdentifier)
}

// TestWriteXarEmptyAndNestedDirs ensures the writer handles empty files and
// nested directories without corrupting the archive (the decoder still parses
// the header/TOC and finds the Distribution).
func TestWriteXarEmptyAndNestedDirs(t *testing.T) {
	root := writeXarTree(t, map[string][]byte{
		"Distribution":                    []byte(testDistribution),
		"base.pkg/empty":                  {},
		"base.pkg/nested/deep/Info.plist": []byte("<plist/>"),
	})
	// An empty directory too (map above only creates dirs with files).
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Resources"), 0o755))

	xarBytes := buildXar(t, root)
	require.ErrorIs(t, file.CheckPKGSignature(bytes.NewReader(xarBytes)), file.ErrNotSigned)

	meta := extractXARMetadata(t, xarBytes)
	require.Equal(t, "Fleet osquery", meta.Name)
}
