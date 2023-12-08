package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231130132931, Down_20231130132931)
}

func Up_20231130132931(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software ADD COLUMN title_id INT(10) UNSIGNED DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add title_id column to software table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software ADD INDEX (title_id)`)
	if err != nil {
		return fmt.Errorf("failed to add title_id index to software table: %w", err)
	}

	return nil
}

func Down_20231130132931(tx *sql.Tx) error {
	return nil
}
