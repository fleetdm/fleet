package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251015103505, Down_20251015103505)
}

func Up_20251015103505(_ *sql.Tx) error {
	// This migration recomputed software checksums for existing 'apps' rows so the
	// checksum would incorporate the software name, using the SQL MD5() function
	// (removed in MySQL 9.6/9.7). It only ever rewrote rows that predated the
	// name-inclusive checksum. A fresh install replays this against an empty
	// software table — and runtime inserts already compute the name-inclusive
	// checksum in Go — so there is nothing to backfill. Existing instances already
	// ran the original backfill.
	return nil
}

func Down_20251015103505(tx *sql.Tx) error {
	return nil
}
