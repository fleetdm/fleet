package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240521143024, Down_20240521143024)
}

func Up_20240521143024(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_installers ADD COLUMN self_service bool NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add self_service to software_installers: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_software_installs ADD COLUMN self_service bool NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add self_service bool to host_software_installs: %w", err)
	}

	return nil
}

func Down_20240521143024(tx *sql.Tx) error {
	return nil
}
