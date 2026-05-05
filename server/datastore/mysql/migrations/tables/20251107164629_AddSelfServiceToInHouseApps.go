package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251107164629, Down_20251107164629)
}

func Up_20251107164629(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE in_house_apps
	ADD COLUMN self_service tinyint(1) NOT NULL DEFAULT '0'
`)
	if err != nil {
		return fmt.Errorf("failed to alter in_house_apps self_service: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_in_house_software_installs
	ADD COLUMN self_service tinyint(1) NOT NULL DEFAULT '0'
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_in_house_software_installs self_service: %w", err)
	}

	return nil
}

func Down_20251107164629(tx *sql.Tx) error {
	return nil
}
