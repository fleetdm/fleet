package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20261015212550, Down_20261015212550)
}

func Up_20261015212550(tx *sql.Tx) error {
	var changes []string
	if !indexExistsTx(tx, "host_vpp_software_installs", "idx_host_vpp_software_installs_version_host_id") {
		changes = append(changes, "ADD INDEX idx_host_vpp_software_installs_version_host_id (vpp_app_team_id, host_id, canceled, removed, id)")
	}
	// Drop the single-column index in the same statement, the new index starts with vpp_app_team_id and backs the foreign key instead
	if indexExistsTx(tx, "host_vpp_software_installs", "fk_host_vpp_software_installs_vpp_app_team_id") {
		changes = append(changes, "DROP INDEX fk_host_vpp_software_installs_vpp_app_team_id")
	}
	if len(changes) == 0 {
		return nil
	}
	_, err := tx.Exec(fmt.Sprintf(`ALTER TABLE host_vpp_software_installs %s, ALGORITHM=INPLACE, LOCK=NONE`, strings.Join(changes, ", ")))
	if err != nil {
		return fmt.Errorf("replacing vpp_app_team_id index on host_vpp_software_installs: %w", err)
	}
	return nil
}

func Down_20261015212550(tx *sql.Tx) error {
	return nil
}
