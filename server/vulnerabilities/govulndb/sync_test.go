package govulndb

import (
	"compress/gzip"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
	"github.com/stretchr/testify/require"
)

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
	t.Run("reads the artifact and ignores other files", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", &Artifact{
			SchemaVersion: "1",
			Modules:       map[string][]Advisory{"github.com/air-verse/air": nil},
		})
		require.NoError(t, os.WriteFile(filepath.Join(dir, "osv-ubuntu-2204-2026-09-10.json.gz"), []byte("not ours"), 0o600))

		artifact, err := loadLatestArtifact(t.Context(), dir)
		require.NoError(t, err)
		require.NotNil(t, artifact)
		require.Contains(t, artifact.Modules, "github.com/air-verse/air")
	})

	t.Run("errors on a corrupted artifact", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "govulndb-2026-09-09.json.gz"), []byte("not gzip"), 0o600))

		_, err := loadLatestArtifact(t.Context(), dir)
		require.Error(t, err)
	})
}

func TestNewestAsset(t *testing.T) {
	asset := func(name string) *vulnrepo.Asset { return &vulnrepo.Asset{Name: name} }

	t.Run("no artifact in the release", func(t *testing.T) {
		// Not an error: the publisher may not have run yet, and that must not fail the cron.
		require.Nil(t, newestAsset(&vulnrepo.Release{Assets: map[string]*vulnrepo.Asset{}}))
	})

	t.Run("newest of several", func(t *testing.T) {
		// Map iteration order is random, so this has to hold whatever order it visits.
		got := newestAsset(&vulnrepo.Release{Assets: map[string]*vulnrepo.Asset{
			"govulndb-2026-09-01.json.gz": asset("govulndb-2026-09-01.json.gz"),
			"govulndb-2026-09-09.json.gz": asset("govulndb-2026-09-09.json.gz"),
			"govulndb-2026-08-25.json.gz": asset("govulndb-2026-08-25.json.gz"),
		}})
		require.NotNil(t, got)
		require.Equal(t, "govulndb-2026-09-09.json.gz", got.Name)
	})
}
