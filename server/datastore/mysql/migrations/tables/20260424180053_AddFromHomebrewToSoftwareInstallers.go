package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260424180053, Down_20260424180053)
}

func Up_20260424180053(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE software_installers
		ADD COLUMN from_homebrew TINYINT(1) NOT NULL DEFAULT 0
	`)
	if err != nil {
		return fmt.Errorf("add from_homebrew to software_installers: %w", err)
	}
	return nil
}

func Down_20260424180053(tx *sql.Tx) error {
	return nil
}
