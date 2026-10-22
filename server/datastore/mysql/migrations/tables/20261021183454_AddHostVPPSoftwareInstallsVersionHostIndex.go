package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261021183454, Down_20261021183454)
}

func Up_20261021183454(tx *sql.Tx) error {
	if indexExistsTx(tx, "host_vpp_software_installs", "idx_host_vpp_software_installs_version_host_id") {
		return nil
	}
	// Drop the single-column index in the same statement, the new index starts with vpp_app_team_id and backs the foreign key instead
	_, err := tx.Exec(`ALTER TABLE host_vpp_software_installs
		ADD INDEX idx_host_vpp_software_installs_version_host_id (vpp_app_team_id, host_id, canceled, removed, id, verification_at, verification_failed_at),
		DROP INDEX fk_host_vpp_software_installs_vpp_app_team_id,
		ALGORITHM=INPLACE, LOCK=NONE`)
	if err != nil {
		return fmt.Errorf("replacing vpp_app_team_id index on host_vpp_software_installs: %w", err)
	}
	return nil
}

func Down_20261021183454(tx *sql.Tx) error {
	return nil
}
