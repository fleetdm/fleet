package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240110134315, Down_20240110134315)
}

func Up_20240110134315(tx *sql.Tx) error {
	addColumnStmt := `
		ALTER TABLE operating_systems
		ADD COLUMN display_version VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';
	`
	if _, err := tx.Exec(addColumnStmt); err != nil {
		return fmt.Errorf("adding operating_systems column: %w", err)
	}

	dropIndexStmt := `
		ALTER TABLE operating_systems
		DROP INDEX idx_unique_os;
	`
	if _, err := tx.Exec(dropIndexStmt); err != nil {
		return fmt.Errorf("dropping operating_systems index: %w", err)
	}

	addIndexStmt := `
		ALTER TABLE operating_systems
		ADD UNIQUE INDEX idx_unique_os (name, version, arch, kernel_version, platform, display_version);
	`
	if _, err := tx.Exec(addIndexStmt); err != nil {
		return fmt.Errorf("adding operating_systems index: %w", err)
	}

	return nil
}

func Down_20240110134315(tx *sql.Tx) error {
	return nil
}
