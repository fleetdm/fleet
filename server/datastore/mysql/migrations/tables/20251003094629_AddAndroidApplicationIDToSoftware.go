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
		return fmt.Errorf("failed to add software_titles.is_kernel column: %w", err)
	}

	// Drop the column to update the definition
	_, err = tx.Exec(`ALTER TABLE software_titles DROP COLUMN unique_identifier`)
	if err != nil {
		return fmt.Errorf("failed to add software_titles.is_kernel column: %w", err)
	}

	_, err = tx.Exec(`
        ALTER TABLE software_titles
        ADD COLUMN unique_identifier VARCHAR(255) GENERATED ALWAYS AS (COALESCE(bundle_identifier, COALESCE(application_id, name))) VIRTUAL;
    `)
	if err != nil {
		return fmt.Errorf("failed to add generated column unique_identifier: %w", err)
	}

	return nil
}

func Down_20251003094629(tx *sql.Tx) error {
	return nil
}
