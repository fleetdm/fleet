package govulndb

import (
	"compress/gzip"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsArtifactAsset(t *testing.T) {
	for name, want := range map[string]bool{
		"govulndb-2026-09-09.json.gz":           true,
		"govulndb-2026-09-09.json":              false,
		"osv-ubuntu-2204-2026-09-09.json.gz":    false,
		"fleet_msrc_Windows_10-2026_09_09.json": false,
		"govulndb.json.gz":                      false,
		"":                                      false,
	} {
		require.Equal(t, want, isArtifactAsset(name), name)
	}
}

// writeArtifact writes a gzipped artifact and returns its path.
func writeArtifact(t *testing.T, dir string, name string, artifact *Artifact) string {
	t.Helper()

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	require.NoError(t, json.MarshalWrite(gz, artifact))
	require.NoError(t, gz.Close())

	return path
}

func TestLoadLatestArtifact(t *testing.T) {
	t.Run("no artifact on disk", func(t *testing.T) {
		artifact, err := loadLatestArtifact(t.TempDir())
		require.NoError(t, err)
		require.Nil(t, artifact)
	})

	t.Run("reads the newest artifact and ignores other files", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-01.json.gz", &Artifact{
			SchemaVersion: "1",
			Modules:       map[string][]Advisory{"old": nil},
		})
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", &Artifact{
			SchemaVersion: "1",
			Modules:       map[string][]Advisory{"new": nil},
		})
		require.NoError(t, os.WriteFile(filepath.Join(dir, "osv-ubuntu-2204-2026-09-10.json.gz"), []byte("not ours"), 0o600))

		artifact, err := loadLatestArtifact(dir)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		require.Contains(t, artifact.Modules, "new")
	})

	t.Run("errors on a corrupted artifact", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "govulndb-2026-09-09.json.gz"), []byte("not gzip"), 0o600))

		_, err := loadLatestArtifact(dir)
		require.Error(t, err)
	})
}

func TestRemoveSupersededArtifacts(t *testing.T) {
	dir := t.TempDir()

	keep := "govulndb-2026-09-09.json.gz"
	for _, name := range []string{keep, "govulndb-2026-09-01.json.gz", "govulndb-2026-08-25.json.gz"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}
	// Artifacts belonging to other data sources share the directory and must survive.
	other := "osv-ubuntu-2204-2026-09-09.json.gz"
	require.NoError(t, os.WriteFile(filepath.Join(dir, other), []byte("x"), 0o600))

	require.NoError(t, removeSupersededArtifacts(dir, keep))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.ElementsMatch(t, []string{keep, other}, names)
}

func TestFileSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))

	digest, err := fileSHA256(path)
	require.NoError(t, err)
	require.Equal(t, "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", digest)
}
