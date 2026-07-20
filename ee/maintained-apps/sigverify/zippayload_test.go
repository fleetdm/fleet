package sigverify

import (
	"archive/zip"
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

	t.Run("invalid zip", func(t *testing.T) {
		notZip := filepath.Join(t.TempDir(), "notzip.zip")
		require.NoError(t, os.WriteFile(notZip, []byte("not a zip"), 0o644))
		_, err := ExtractZipPayload(notZip, t.TempDir(), []string{".msi"})
		require.ErrorContains(t, err, "opening zip")
	})
}
