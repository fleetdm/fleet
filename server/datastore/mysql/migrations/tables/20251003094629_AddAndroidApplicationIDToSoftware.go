package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251003094629, Down_20251003094629)
}

func Up_20251003094629(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_titles ADD COLUMN application_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add software_titles.application_id column: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software ADD COLUMN application_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add software.application_id column: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		DROP INDEX idx_unique_sw_titles
	`); err != nil {
		return fmt.Errorf("failed to drop unique index: %w", err)
	}

	// Drop the column to update the definition
	_, err = tx.Exec(`ALTER TABLE software_titles DROP COLUMN unique_identifier`)
	if err != nil {
		return fmt.Errorf("failed to drop software_titles.unique_identifier column: %w", err)
	}

	_, err = tx.Exec(`
        ALTER TABLE software_titles
        ADD COLUMN unique_identifier VARCHAR(255) GENERATED ALWAYS AS (COALESCE(bundle_identifier, application_id, name)) VIRTUAL;
    `)
	if err != nil {
		return fmt.Errorf("failed to add generated column unique_identifier: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		ADD UNIQUE INDEX idx_unique_sw_titles (unique_identifier, source, extension_for);
	`); err != nil {
		return fmt.Errorf("failed to add unique index: %w", err)
	}

	return nil
}

func Down_20251003094629(tx *sql.Tx) error {
	return nil
}
