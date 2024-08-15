package goval_dictionary

import (
	"database/sql"
	"testing"
)

func TestDatabase(t *testing.T) {
	// build minimal slice of goval-dictionary sqlite schema
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE packages (name TEXT NOT NULL, arch TEXT NOT NULL, version TEXT NOT NULL, definition_id INTEGER NOT NULL)")
	db.Exec("CREATE TABLE cves (cve_id TEXT NOT NULL, advisory_id INTEGER NOT NULL)")
	db.Exec("CREATE TABLE definitions (id INTEGER NOT NULL PRIMARY KEY)")
	db.Exec("CREATE TABLE advisories (id INTEGER NOT NULL PRIMARY KEY, definition_id INTEGER NOT NULL)")

	// TODO populate goval-dictionary sqlite schema with a few vulnerabilities

	t.Run("Non-matching architecture", func(t *testing.T) {
		// TODO
	})

	t.Run("Non-matching package name", func(t *testing.T) {
		// TODO
	})

	t.Run("Fixed version", func(t *testing.T) {
		// TODO
	})

	t.Run("Newer than fixed version", func(t *testing.T) {
		// TODO
	})

	t.Run("Older than fixed version", func(t *testing.T) {
		// TODO
	})

	t.Run("Multiple packages, fixed version", func(t *testing.T) {
		// TODO
	})

	t.Run("Multiple packages, multiple vulnerabilities", func(t *testing.T) {
		// TODO
	})
}
