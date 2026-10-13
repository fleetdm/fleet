package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261013161954, Down_20261013161954)
}

func Up_20261013161954(tx *sql.Tx) error {
	const batchSize = 1000

	var maxInstallID sql.NullInt64
	err := tx.QueryRow(`SELECT MAX(id) FROM host_vpp_software_installs`).Scan(&maxInstallID)
	if err != nil {
		return fmt.Errorf("selecting max host_vpp_software_installs id: %w", err)
	}

	// Batch by id range, MySQL does not allow LIMIT on an UPDATE with a JOIN
	for startID := int64(0); startID < maxInstallID.Int64; startID += batchSize {
		_, err = tx.Exec(`
UPDATE host_vpp_software_installs hvsi
JOIN nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
SET hvsi.verification_failed_at = ncr.updated_at
WHERE hvsi.verification_at IS NULL
	AND hvsi.verification_failed_at IS NULL
	AND hvsi.canceled = 0
	AND ncr.status IN ('Error', 'CommandFormatError')
	AND hvsi.id > ? AND hvsi.id <= ?`, startID, startID+batchSize)
		if err != nil {
			return fmt.Errorf("set verification_failed_at on vpp app installs that errored after id %d: %w", startID, err)
		}
	}
	return nil
}

func Down_20261013161954(tx *sql.Tx) error {
	return nil
}
