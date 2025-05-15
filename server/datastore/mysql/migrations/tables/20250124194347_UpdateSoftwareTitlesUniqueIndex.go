package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250124194347, Down_20250124194347)
}

func Up_20250124194347(tx *sql.Tx) error {
	if _, err := tx.Exec(`
        ALTER TABLE software_titles
        ADD COLUMN unique_identifier VARCHAR(255) GENERATED ALWAYS AS (COALESCE(bundle_identifier, name)) VIRTUAL;
    `); err != nil {
		return fmt.Errorf("failed to add generated column unique_identifier: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		ADD UNIQUE INDEX idx_unique_sw_titles (unique_identifier, source, browser);
	`); err != nil {
		return fmt.Errorf("failed to add unique index: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		DROP INDEX idx_sw_titles,
		ADD INDEX idx_sw_titles (name, source, browser);
	`); err != nil {
		return fmt.Errorf("failed to re-add idx_sw_titles: %w", err)
	}

	return nil
}

func Down_20250124194347(tx *sql.Tx) error {
	return nil
}
