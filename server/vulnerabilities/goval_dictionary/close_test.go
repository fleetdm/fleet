package goval_dictionary

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestDatabaseCloseReleasesFileHandle is the regression test for the
// FD leak in fleetdm/fleet#42741. It exercises the exact lifecycle
// that Analyze depends on: open the sqlite file, run a goval-style
// query (which forces the connection pool to allocate a real file
// descriptor), then Close (which must drain the pool so the FD is
// released before the next vuln refresh atomically replaces the file).
func TestDatabaseCloseReleasesFileHandle(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "repro42741*.sqlite3")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	path := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	defer os.Remove(path)

	// Seed the goval schema so Verify's query succeeds.
	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("seed open: %v", err)
	}
	schema := `
		CREATE TABLE packages (id INTEGER, name TEXT, version TEXT, arch TEXT, definition_id INTEGER);
		CREATE TABLE definitions (id INTEGER);
		CREATE TABLE advisories (id INTEGER, definition_id INTEGER);
		CREATE TABLE cves (id INTEGER, cve_id TEXT, advisory_id INTEGER);`
	if _, err := seed.Exec(schema); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("seed close: %v", err)
	}

	// Open the file the way LoadDb does in production.
	sqlite, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlite.Close() })
	db := Database{sqlite: sqlite}

	// Run a real query to force the pool to allocate a connection on
	// the sqlite file. Matches db.Eval's behavior inside Analyze.
	if err := db.Verfiy(); err != nil {
		t.Fatalf("Verfiy: %v", err)
	}
	if n := sqlite.Stats().OpenConnections; n < 1 {
		t.Fatalf("query should have opened a pool connection, got %d", n)
	}

	// This is the fix: Close must drain the pool so the FD is released.
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if n := sqlite.Stats().OpenConnections; n != 0 {
		t.Fatalf("Close did not drain pool, got %d open connections", n)
	}
	if err := sqlite.Ping(); err == nil {
		t.Fatal("pool still accepting queries after Close; expected error")
	}
}
