package goval_dictionary

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/stretchr/testify/require"
)

// TestLoadDbRejectsEmptyDatabase is the regression test for
// https://github.com/fleetdm/fleet/issues/45602: a goval-dictionary sqlite database
// with no definitions (e.g., a corrupted download) must NOT silently succeed —
// otherwise every existing goval vulnerability for the platform would be marked as
// remediated. LoadDb must return an error in that case so the cron skips the analysis
// (and therefore the deletes that would happen during it).
func TestLoadDbRejectsEmptyDatabase(t *testing.T) {
	platform := oval.NewPlatform("amzn", "Amazon Linux 2 (Karoo)")
	require.True(t, platform.IsGovalDictionarySupported(), "test platform must be supported")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, platform.ToGovalDictionaryFilename())

	// Seed an empty goval schema — same tables as a real database, but with no rows.
	seed, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	schema := `
		CREATE TABLE packages (id INTEGER, name TEXT, version TEXT, arch TEXT, definition_id INTEGER);
		CREATE TABLE definitions (id INTEGER);
		CREATE TABLE advisories (id INTEGER, definition_id INTEGER);
		CREATE TABLE cves (id INTEGER, cve_id TEXT, advisory_id INTEGER);`
	_, err = seed.Exec(schema)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	_, err = LoadDb(platform, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no definitions")

	// File should still exist — LoadDb must not delete it.
	_, statErr := os.Stat(dbPath)
	require.NoError(t, statErr)
}

func TestLoadDbAcceptsNonEmptyDatabase(t *testing.T) {
	platform := oval.NewPlatform("amzn", "Amazon Linux 2 (Karoo)")
	require.True(t, platform.IsGovalDictionarySupported())

	dir := t.TempDir()
	dbPath := filepath.Join(dir, platform.ToGovalDictionaryFilename())

	seed, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	schema := `
		CREATE TABLE packages (id INTEGER, name TEXT, version TEXT, arch TEXT, definition_id INTEGER);
		CREATE TABLE definitions (id INTEGER);
		CREATE TABLE advisories (id INTEGER, definition_id INTEGER);
		CREATE TABLE cves (id INTEGER, cve_id TEXT, advisory_id INTEGER);
		INSERT INTO definitions (id) VALUES (1);`
	_, err = seed.Exec(schema)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	db, err := LoadDb(platform, dir)
	require.NoError(t, err)
	require.NotNil(t, db)
	require.NoError(t, db.Close())
}
