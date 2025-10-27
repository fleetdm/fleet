package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251027164003, Down_20251027164003)
}

func Up_20251027164003(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE in_house_apps
	ADD COLUMN self_service tinyint(1) NOT NULL DEFAULT '0'
`)
	if err != nil {
		return fmt.Errorf("failed to alter in_house_apps self_service: %w", err)
	}
	return nil
}

func Down_20251027164003(tx *sql.Tx) error {
	return nil
}
