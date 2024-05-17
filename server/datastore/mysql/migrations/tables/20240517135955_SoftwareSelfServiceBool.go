package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240517135955, Down_20240517135955)
}

func Up_20240517135955(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_installers ADD COLUMN self_service bool NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add self_service to software_installers")
	}
	return nil
}

func Down_20240517135955(tx *sql.Tx) error {
	return nil
}
