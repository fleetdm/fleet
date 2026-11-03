package vulnrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o644))

	// echo -n "test content" | sha256sum
	digest, err := FileSHA256(testFile)
	require.NoError(t, err)
	require.Equal(t, "sha256:6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72", digest)

	// The prefix has to match the format Asset.Digest carries, since Refresh compares the
	// two directly; a mismatch would silently re-download every artifact on every run.
	_, err = FileSHA256(filepath.Join(tmpDir, "nonexistent.txt"))
	require.Error(t, err)
}

func TestIsPlainFileName(t *testing.T) {
	for name, want := range map[string]bool{
		"govulndb-2026-09-09.json.gz":        true,
		"osv-ubuntu-2204-2026-09-09.json.gz": true,
		"govulndb-../../x.json.gz":           false,
		"../govulndb-2026-09-09.json.gz":     false,
		"nested/govulndb-2026-09-09.json.gz": false,
		"/etc/passwd":                        false,
		"..":                                 false,
		"":                                   false,
	} {
		require.Equalf(t, want, isPlainFileName(name), "isPlainFileName(%q)", name)
	}
}

func TestNewestLocalAsset(t *testing.T) {
	isGoVulnDB := func(name string) bool {
		return strings.HasPrefix(name, "govulndb-") && strings.HasSuffix(name, ".json.gz")
	}

	t.Run("no matching file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "osv-ubuntu-2204-2026-09-09.json.gz"), []byte("x"), 0o600))

		got, err := NewestLocalAsset(dir, isGoVulnDB)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("missing directory", func(t *testing.T) {
		_, err := NewestLocalAsset(filepath.Join(t.TempDir(), "nope"), isGoVulnDB)
		require.Error(t, err)
	})

	t.Run("newest by name, not by modification time", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{
			"govulndb-2026-09-01.json.gz",
			"govulndb-2026-09-09.json.gz",
			"govulndb-2026-08-25.json.gz",
			"osv-ubuntu-2204-2026-09-10.json.gz",
		} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
		}
		// A matching name that is a directory is never a candidate.
		require.NoError(t, os.Mkdir(filepath.Join(dir, "govulndb-2026-09-30.json.gz"), 0o700))
		// A re-download of an old artifact touches its mtime; the date in the name still wins.
		old := time.Now().Add(-48 * time.Hour)
		require.NoError(t, os.Chtimes(filepath.Join(dir, "govulndb-2026-09-09.json.gz"), old, old))

		got, err := NewestLocalAsset(dir, isGoVulnDB)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(dir, "govulndb-2026-09-09.json.gz"), got)
	})
}

func TestRemoveLocalAssets(t *testing.T) {
	dir := t.TempDir()
	keep := "govulndb-2026-09-09.json.gz"
	for _, name := range []string{keep, "govulndb-2026-09-01.json.gz", "govulndb-2026-08-25.json.gz", "osv-ubuntu-2204-2026-09-09.json.gz"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, "govulndb-2026-07-01.json.gz"), 0o700))

	err := RemoveLocalAssets(dir, func(name string) bool {
		return strings.HasPrefix(name, "govulndb-") && name != keep
	})
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	// Other data sources share the directory and must survive, and directories are never removed.
	require.ElementsMatch(t, []string{keep, "osv-ubuntu-2204-2026-09-09.json.gz", "govulndb-2026-07-01.json.gz"}, names)

	require.Error(t, RemoveLocalAssets(filepath.Join(dir, "nope"), func(string) bool { return true }))
}
