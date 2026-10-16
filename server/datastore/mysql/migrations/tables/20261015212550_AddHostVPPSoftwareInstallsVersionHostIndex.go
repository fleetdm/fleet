package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261015212550, Down_20261015212550)
}

func Up_20261015212550(tx *sql.Tx) error {
	if indexExistsTx(tx, "host_vpp_software_installs", "idx_host_vpp_software_installs_version_host_id") {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE host_vpp_software_installs ADD INDEX idx_host_vpp_software_installs_version_host_id (vpp_app_team_id, host_id, canceled, removed, id), ALGORITHM=INPLACE, LOCK=NONE`)
	if err != nil {
		return fmt.Errorf("adding idx_host_vpp_software_installs_version_host_id to host_vpp_software_installs: %w", err)
	}
	return nil
}

func Down_20261015212550(tx *sql.Tx) error {
	return nil
}
