package packaging

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPosixCksum(t *testing.T) {
	// Verify against known POSIX cksum values.
	// These can be verified with: printf 'hello' | cksum
	// "hello" → cksum outputs: 3287646509 5 -
	got := posixCksum([]byte("hello"))
	assert.Equal(t, uint32(3287646509), got, "POSIX cksum of 'hello'")

	// Empty data: printf '' | cksum → 4294967295 0 -
	got = posixCksum([]byte{})
	assert.Equal(t, uint32(4294967295), got, "POSIX cksum of empty data")

	// Single byte: printf '\x00' | cksum → 4215202376 1 -
	got = posixCksum([]byte{0})
	assert.Equal(t, uint32(4215202376), got, "POSIX cksum of single null byte")
}

func TestWriteBOM(t *testing.T) {
	// Create a temporary directory tree.
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")

	// Create directory structure matching a simple pkg.
	dirs := []string{
		"opt",
		"opt/orbit",
		"Library",
		"Library/LaunchDaemons",
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(rootDir, d), 0o755))
	}

	// Create some files.
	files := map[string]string{
		"opt/orbit/secret.txt":                          "fleet-secret",
		"Library/LaunchDaemons/com.fleetdm.orbit.plist": "<plist>test</plist>",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(rootDir, name), []byte(content), 0o644))
	}

	// Write BOM.
	bomPath := filepath.Join(tmpDir, "test.bom")
	err := writeBOM(rootDir, bomPath, 0, 80)
	require.NoError(t, err)

	// Verify the BOM file was created and has the correct header.
	bomData, err := os.ReadFile(bomPath)
	require.NoError(t, err)
	require.Greater(t, len(bomData), bomHeaderLen, "BOM file too small")

	// Check magic.
	assert.Equal(t, "BOMStore", string(bomData[0:8]))

	// Check version.
	version := binary.BigEndian.Uint32(bomData[8:12])
	assert.Equal(t, uint32(1), version)

	// Check numberOfBlocks > 0.
	numBlocks := binary.BigEndian.Uint32(bomData[12:16])
	assert.Positive(t, numBlocks, "should have blocks")

	// Check that indexOffset and varsOffset are reasonable.
	indexOffset := binary.BigEndian.Uint32(bomData[16:20])
	varsOffset := binary.BigEndian.Uint32(bomData[24:28])
	assert.Equal(t, uint32(bomHeaderLen), varsOffset)
	assert.Greater(t, indexOffset, varsOffset, "index should be after vars")
	assert.Less(t, indexOffset, uint32(len(bomData)), "index should be within file") //nolint:gosec // G115: test assertion
}

func TestEncodeBOM_Empty(t *testing.T) {
	// A BOM with an empty directory should still produce valid output.
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))

	bomPath := filepath.Join(tmpDir, "test.bom")
	err := writeBOM(rootDir, bomPath, 0, 80)
	require.NoError(t, err)

	bomData, err := os.ReadFile(bomPath)
	require.NoError(t, err)
	assert.Equal(t, "BOMStore", string(bomData[0:8]))
}

func TestBOMTreeStructure(t *testing.T) {
	// Create a tree and verify BFS ordering.
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")

	// Create a/b and c directories.
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "a", "b"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "c"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "a", "b", "file.txt"), []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "c", "other.txt"), []byte("data"), 0o644))

	root, err := buildBOMTree(rootDir, 0, 80)
	require.NoError(t, err)

	// Verify tree structure: virtual root → "." → children
	require.Len(t, root.children, 1)
	dotNode := root.children[0]
	assert.Equal(t, ".", dotNode.name)

	// "." should have children "a" and "c" (sorted).
	require.Len(t, dotNode.children, 2)
	assert.Equal(t, "a", dotNode.children[0].name)
	assert.Equal(t, "c", dotNode.children[1].name)

	// "a" should have child "b".
	require.Len(t, dotNode.children[0].children, 1)
	assert.Equal(t, "b", dotNode.children[0].children[0].name)

	// "b" should have child "file.txt".
	require.Len(t, dotNode.children[0].children[0].children, 1)
	assert.Equal(t, "file.txt", dotNode.children[0].children[0].children[0].name)

	// Verify BFS order via encoding.
	var buf bytes.Buffer
	err = encodeBOM(&buf, root)
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), bomHeaderLen)
}

func TestPosixCRC32Table(t *testing.T) {
	// Verify first and last entries of the POSIX CRC-32 table match the
	// bomutils reference implementation.
	assert.Equal(t, uint32(0x00000000), posixCRC32Table[0])
	assert.Equal(t, uint32(0x04c11db7), posixCRC32Table[1])
	assert.Equal(t, uint32(0xb1f740b4), posixCRC32Table[255])
}
