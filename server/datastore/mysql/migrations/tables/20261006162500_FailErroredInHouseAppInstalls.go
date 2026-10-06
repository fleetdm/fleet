package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261006162500, Down_20261006162500)
}

func Up_20261006162500(tx *sql.Tx) error {
	if _, err := tx.Exec(`
UPDATE host_in_house_software_installs hihsi
JOIN hosts h ON h.id = hihsi.host_id
JOIN nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hihsi.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE hihsi.verification_at IS NULL
	AND hihsi.verification_failed_at IS NULL
	AND hihsi.canceled = 0
	AND ncr.status = 'Error'`); err != nil {
		return fmt.Errorf("set verification_failed_at on in-house app installs that errored: %w", err)
	}
	return nil
}

func Down_20261006162500(tx *sql.Tx) error {
	return nil
}
