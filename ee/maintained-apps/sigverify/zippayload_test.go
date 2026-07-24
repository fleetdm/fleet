package sigverify

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "installer.zip")
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		entry, err := w.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return zipPath
}

func TestExtractZipPayload(t *testing.T) {
	t.Run("top-level msi", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{
			"README.txt": "readme",
			"RealVNC-Connect-Viewer-8.4.1-Windows.msi": "msi bytes",
		})
		dest := t.TempDir()
		payload, err := ExtractZipPayload(zipPath, dest, []string{".msi", ".exe"})
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dest, "RealVNC-Connect-Viewer-8.4.1-Windows.msi"), payload)
		content, err := os.ReadFile(payload)
		require.NoError(t, err)
		require.Equal(t, "msi bytes", string(content))
	})

	t.Run("nested exe, msi preferred over exe", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{
			"app/setup.exe":      "exe bytes",
			"app/installer.msi":  "msi bytes",
			"app/data/other.txt": "other",
		})
		dest := t.TempDir()
		payload, err := ExtractZipPayload(zipPath, dest, []string{".msi", ".exe"})
		require.NoError(t, err)
		// .msi listed first in exts, so it wins even though both exist.
		require.Equal(t, filepath.Join(dest, "app", "installer.msi"), payload)
	})

	t.Run("no payload", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{"README.txt": "readme"})
		payload, err := ExtractZipPayload(zipPath, t.TempDir(), []string{".msi", ".exe"})
		require.NoError(t, err)
		require.Empty(t, payload)
	})

	t.Run("zip-slip entries are skipped", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{
			"../evil.msi":  "evil",
			"good/app.msi": "good",
		})
		dest := t.TempDir()
		payload, err := ExtractZipPayload(zipPath, dest, []string{".msi"})
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dest, "good", "app.msi"), payload)
		// The traversal entry must not have escaped destDir.
		_, statErr := os.Stat(filepath.Join(filepath.Dir(dest), "evil.msi"))
		require.True(t, os.IsNotExist(statErr))
	})

	t.Run("only the payload entry is extracted", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{
			"big-other-file.dat": "must not be written to disk",
			"docs/readme.txt":    "must not be written to disk",
			"Setup.MSI":          "msi bytes", // uppercase extension still matches
		})
		dest := t.TempDir()
		payload, err := ExtractZipPayload(zipPath, dest, []string{".msi", ".exe"})
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dest, "Setup.MSI"), payload)
		_, statErr := os.Stat(filepath.Join(dest, "big-other-file.dat"))
		require.True(t, os.IsNotExist(statErr))
		_, statErr = os.Stat(filepath.Join(dest, "docs"))
		require.True(t, os.IsNotExist(statErr))
	})

	t.Run("payload deeper than one directory is not considered", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{"a/b/c/installer.msi": "too deep"})
		payload, err := ExtractZipPayload(zipPath, t.TempDir(), []string{".msi"})
		require.NoError(t, err)
		require.Empty(t, payload)
	})

	t.Run("entry size limit boundary", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{"payload.msi": "1234567890"}) // 10 bytes
		r, err := zip.OpenReader(zipPath)
		require.NoError(t, err)
		defer r.Close()
		entry := r.File[0]

		// An entry exactly at the limit extracts intact.
		target := filepath.Join(t.TempDir(), "at-limit.msi")
		require.NoError(t, extractZipEntry(entry, target, 10))
		content, err := os.ReadFile(target)
		require.NoError(t, err)
		require.Equal(t, "1234567890", string(content))

		// One byte over the limit is rejected from the declared size, before
		// anything is written to disk.
		overTarget := filepath.Join(t.TempDir(), "over-limit.msi")
		err = extractZipEntry(entry, overTarget, 9)
		require.ErrorContains(t, err, "declares 10 bytes, exceeding the 9 byte extraction limit")
		_, statErr := os.Stat(overTarget)
		require.True(t, os.IsNotExist(statErr))
	})

	t.Run("lying declared size still fails extraction", func(t *testing.T) {
		// A header that under-declares the uncompressed size passes the
		// declared-size check, but extraction must still fail rather than
		// yield a silently truncated payload: archive/zip errors when the
		// stream exceeds the declared size, and the streaming limit in
		// extractZipEntry remains as a further wall.
		var deflated bytes.Buffer
		fw, err := flate.NewWriter(&deflated, flate.DefaultCompression)
		require.NoError(t, err)
		_, err = fw.Write([]byte("1234567890"))
		require.NoError(t, err)
		require.NoError(t, fw.Close())

		zipPath := filepath.Join(t.TempDir(), "lying.zip")
		f, err := os.Create(zipPath)
		require.NoError(t, err)
		zw := zip.NewWriter(f)
		w, err := zw.CreateRaw(&zip.FileHeader{
			Name:               "lying.msi",
			Method:             zip.Deflate,
			CRC32:              crc32.ChecksumIEEE([]byte("1234567890")),
			CompressedSize64:   uint64(deflated.Len()), // nolint:gosec // buffer length is never negative
			UncompressedSize64: 5,                      // lies: the stream inflates to 10 bytes
		})
		require.NoError(t, err)
		_, err = w.Write(deflated.Bytes())
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		require.NoError(t, f.Close())

		r, err := zip.OpenReader(zipPath)
		require.NoError(t, err)
		defer r.Close()
		err = extractZipEntry(r.File[0], filepath.Join(t.TempDir(), "out.msi"), 5)
		require.ErrorContains(t, err, "extracting lying.msi")
	})

	t.Run("invalid zip", func(t *testing.T) {
		notZip := filepath.Join(t.TempDir(), "notzip.zip")
		require.NoError(t, os.WriteFile(notZip, []byte("not a zip"), 0o644))
		_, err := ExtractZipPayload(notZip, t.TempDir(), []string{".msi"})
		require.ErrorContains(t, err, "opening zip")
	})
}

// writeRawEntryZip writes a zip whose entries carry the given declared
// uncompressed sizes without writing any actual data (CreateRaw trusts the
// header), so tests can declare absurd sizes cheaply.
func writeRawEntryZip(t *testing.T, declaredSizes []uint64) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "declared.zip")
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	for i, size := range declaredSizes {
		_, err := zw.CreateRaw(&zip.FileHeader{
			Name:               fmt.Sprintf("entry-%d.bin", i),
			Method:             zip.Store,
			CompressedSize64:   0,
			UncompressedSize64: size,
		})
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return zipPath
}

func TestPreflightZip(t *testing.T) {
	t.Run("normal archive passes", func(t *testing.T) {
		zipPath := writeTestZip(t, map[string]string{"App.app/Contents/MacOS/app": "binary", "readme.txt": "hi"})
		require.NoError(t, PreflightZip(zipPath))
	})

	t.Run("oversized declared entry", func(t *testing.T) {
		zipPath := writeRawEntryZip(t, []uint64{maxZipEntrySize + 1})
		require.ErrorContains(t, PreflightZip(zipPath), "byte extraction limit")
	})

	t.Run("total declared size over limit", func(t *testing.T) {
		nine := uint64(9 << 30)
		zipPath := writeRawEntryZip(t, []uint64{nine, nine, nine, nine}) // 36 GiB total
		require.ErrorContains(t, PreflightZip(zipPath), "total uncompressed bytes")
	})

	t.Run("invalid zip", func(t *testing.T) {
		notZip := filepath.Join(t.TempDir(), "notzip.zip")
		require.NoError(t, os.WriteFile(notZip, []byte("not a zip"), 0o644))
		require.ErrorContains(t, PreflightZip(notZip), "opening zip")
	})
}
