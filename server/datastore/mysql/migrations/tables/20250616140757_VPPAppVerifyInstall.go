package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250616140757, Down_20250616140757)
}

func Up_20250616140757(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE host_vpp_software_installs
		ADD COLUMN verification_command_uuid VARCHAR(127) NULL,
		ADD COLUMN verification_at DATETIME(6) NULL,
		ADD COLUMN verification_failed_at DATETIME(6) NULL
		`)
	if err != nil {
		return fmt.Errorf("failed to add host_vpp_software_installs.verification_command_uuid: %w", err)
	}

	return nil
}

func Down_20250616140757(tx *sql.Tx) error {
	return nil
}
