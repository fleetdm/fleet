package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20250624140757, Down_20250624140757)
}

func Up_20250624140757(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE host_vpp_software_installs
		ADD COLUMN verification_command_uuid VARCHAR(127) NULL,
		ADD COLUMN verification_at DATETIME(6) NULL,
		ADD COLUMN verification_failed_at DATETIME(6) NULL,
		ADD INDEX idx_host_vpp_software_installs_verification_at (verification_at),
		ADD INDEX idx_host_vpp_software_installs_verification_failed_at (verification_failed_at)
		`)
	if err != nil {
		return fmt.Errorf("failed to add host_vpp_software_installs.verification_command_uuid: %w", err)
	}

	_, err = tx.Exec(`
UPDATE
	host_vpp_software_installs hvsi
	INNER JOIN nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
SET
	hvsi.verification_at = IF(ncr.status = 'Acknowledged', CURRENT_TIMESTAMP(6), NULL),
	hvsi.verification_failed_at = IF(ncr.status = ? OR ncr.status = ?, CURRENT_TIMESTAMP(6), NULL);
	`, fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError)
	if err != nil {
		return fmt.Errorf("failed to set existing vpp install verification statuses: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE jobs
ADD INDEX idx_jobs_name_state (name,state)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index on jobs table: %w", err)
	}

	return nil
}

func Down_20250624140757(tx *sql.Tx) error {
	return nil
}
