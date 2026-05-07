package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260507193201, Down_20260507193201)
}

func Up_20260507193201(tx *sql.Tx) error {
	// from_homebrew stores the homebrew cask token this installer was imported from
	// (e.g. "firefox"). Empty string means the installer was not imported from homebrew,
	// so the column doubles as a boolean flag for the UI while also preserving enough
	// provenance for the homebrew_updates cron to look up the cask again.
	_, err := tx.Exec(`
		ALTER TABLE software_installers
		ADD COLUMN from_homebrew VARCHAR(255) NOT NULL DEFAULT ''
	`)
	if err != nil {
		return fmt.Errorf("add from_homebrew to software_installers: %w", err)
	}
	return nil
}

func Down_20260507193201(tx *sql.Tx) error {
	return nil
}
