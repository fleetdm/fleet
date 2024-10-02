package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241002210000, Down_20241002210000)
}

func Up_20241002210000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE policies
		ADD COLUMN script_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_policies_script_id (script_id) REFERENCES scripts (id);
	`); err != nil {
		return fmt.Errorf("failed to add script_id to policies: %w", err)
	}

	return nil
}

func Down_20241002210000(tx *sql.Tx) error {
	return nil
}
