package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260902150237, Down_20260902150237)
}

func Up_20260902150237(tx *sql.Tx) error {
	stmt := `ALTER TABLE host_vpp_software_installs
		ADD INDEX idx_host_vpp_software_installs_host_adam_platform_created (host_id, adam_id, platform, created_at),
		ALGORITHM=INPLACE, LOCK=NONE`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to add idx_host_vpp_software_installs_host_adam_platform_created: %w", err)
	}

	stmt = `ALTER TABLE host_in_house_software_installs
		ADD INDEX idx_host_in_house_software_installs_host_app_created (host_id, in_house_app_id, created_at),
		ALGORITHM=INPLACE, LOCK=NONE`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to add idx_host_in_house_software_installs_host_app_created: %w", err)
	}
	return nil
}

func Down_20260902150237(tx *sql.Tx) error {
	return nil
}
