package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260608210432, Down_20260608210432)
}

func Up_20260608210432(tx *sql.Tx) error {
	// Some macOS apps report the sentinel last_opened_time 315532800.0
	// (1980-01-01 UTC, the DOS/FAT epoch) for apps that were never opened. Older
	// Fleet versions stored this as a genuine timestamp, which the UI then
	// rendered as a date decades in the past instead of "Never". Clear only this
	// known sentinel value to NULL. We deliberately avoid a broad date cutoff
	// here: last_opened_at is shared with non-macOS sources (e.g. Linux deb/rpm
	// values derived from file atime), where legitimate pre-2001 timestamps can
	// occur and must not be wiped.
	if _, err := tx.Exec(`
		UPDATE host_software
		SET last_opened_at = NULL
		WHERE last_opened_at = '1980-01-01 00:00:00'
	`); err != nil {
		return fmt.Errorf("clearing sentinel host_software.last_opened_at values: %w", err)
	}
	return nil
}

func Down_20260608210432(tx *sql.Tx) error {
	return nil
}
