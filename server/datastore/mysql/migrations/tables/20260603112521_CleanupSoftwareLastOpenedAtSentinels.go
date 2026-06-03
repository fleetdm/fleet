package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260603112521, Down_20260603112521)
}

func Up_20260603112521(tx *sql.Tx) error {
	// Some macOS apps report sentinel last_opened_time values (e.g. -1.0 or
	// 315532800.0, the 1980-01-01 UTC DOS/FAT epoch) for apps that were never
	// opened. Older Fleet versions stored these as genuine timestamps, which the
	// UI then rendered as a date decades in the past instead of "Never". No app
	// could have been opened before macOS existed, so clear any last_opened_at
	// before 2001-01-01 UTC (roughly when macOS X was released) to NULL.
	if _, err := tx.Exec(`
		UPDATE host_software
		SET last_opened_at = NULL
		WHERE last_opened_at IS NOT NULL
		  AND last_opened_at < '2001-01-01 00:00:00'
	`); err != nil {
		return fmt.Errorf("clearing sentinel host_software.last_opened_at values: %w", err)
	}
	return nil
}

func Down_20260603112521(tx *sql.Tx) error {
	return nil
}
