package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251024145618, Down_20251024145618)
}

func Up_20251024145618(tx *sql.Tx) error {
	// CHAR(38) to account for 32 hex chars + 4 hyphens + open/close curly braces
	_, err := tx.Exec(`ALTER TABLE software_titles ADD COLUMN upgrade_code CHAR(38) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add software_titles.upgrade_code column: %w", err)
	}
	_, err = tx.Exec(`UPDATE software_titles SET upgrade_code = '' WHERE source = 'programs'`)
	if err != nil {
		return fmt.Errorf("failed to add default empty string value to software_titles.upgrade_code column for rows where source = 'programs': %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software ADD COLUMN upgrade_code CHAR(38) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add software.upgrade_code column: %w", err)
	}

	_, err = tx.Exec(`UPDATE software SET upgrade_code = '' WHERE source = 'programs'`)
	if err != nil {
		return fmt.Errorf("failed to add default empty string value to software.upgrade_code column for rows where source == 'programs': %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_titles MODIFY COLUMN unique_identifier VARCHAR(255) GENERATED ALWAYS AS (COALESCE(bundle_identifier, application_id, upgrade_code, name)) VIRTUAL`)
	if err != nil {
		return fmt.Errorf("failed to alter definition of software_titles.unique_identifier column to include upgrade_code in its COALESCE: %w", err)
	}

	return nil
}

func Down_20251024145618(tx *sql.Tx) error {
	return nil
}
