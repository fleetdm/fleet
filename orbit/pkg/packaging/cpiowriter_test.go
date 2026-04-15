package packaging

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCPIOGzip(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")

	// Create source directory with files.
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("hello world"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested"), 0o644))

	dstPath := filepath.Join(tmpDir, "output.cpio.gz")
	err := writeCPIOGzip(srcDir, dstPath, 0, 80)
	require.NoError(t, err)

	// Verify the output is valid gzip.
	f, err := os.Open(dstPath)
	require.NoError(t, err)
	defer f.Close()

	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()

	data, err := io.ReadAll(gr)
	require.NoError(t, err)

	// The data should start with a CPIO ODC magic.
	assert.Greater(t, len(data), 76, "cpio data too small")
	assert.Equal(t, "070707", string(data[0:6]), "CPIO ODC magic")

	// Should contain TRAILER!!!
	assert.True(t, bytes.Contains(data, []byte("TRAILER!!!")))
}

func TestWriteCPIO_ODCFormat(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")

	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("abc"), 0o644))

	var buf bytes.Buffer
	err := writeCPIO(&buf, srcDir, 0, 80)
	require.NoError(t, err)

	data := buf.String()

	// Should start with magic.
	assert.True(t, strings.HasPrefix(data, "070707"))

	// Count number of 070707 magic sequences (should be 3: ".", "./test.txt", TRAILER!!!).
	count := strings.Count(data, "070707")
	assert.Equal(t, 3, count, "should have 3 entries: '.', './test.txt', and TRAILER!!!")

	// Verify TRAILER is present.
	assert.Contains(t, data, "TRAILER!!!")
}

func TestWriteCPIO_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	var buf bytes.Buffer
	err := writeCPIO(&buf, srcDir, 0, 80)
	require.NoError(t, err)

	data := buf.String()
	// Should have "." entry and TRAILER.
	count := strings.Count(data, "070707")
	assert.Equal(t, 2, count, "should have 2 entries: '.' and TRAILER!!!")
}

func TestWriteCPIO_UIDGIDInHeader(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("x"), 0o644))

	var buf bytes.Buffer
	err := writeCPIO(&buf, srcDir, 0, 80)
	require.NoError(t, err)

	data := buf.Bytes()
	// ODC header layout: magic(6) + dev(6) + ino(6) + mode(6) + uid(6) + gid(6)
	// First entry is ".":
	// uid starts at offset 24, gid at offset 30.
	require.Greater(t, len(data), 76)
	uid := string(data[24:30])
	gid := string(data[30:36])
	assert.Equal(t, "000000", uid, "uid should be 0")
	assert.Equal(t, "000120", gid, "gid should be 80 (0120 octal)")
}
