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

	_, err = tx.Exec(`ALTER TABLE software ADD COLUMN upgrade_code CHAR(38) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add software.upgrade_code column: %w", err)
	}
	return nil
}

func Down_20251024145618(tx *sql.Tx) error {
	return nil
}
