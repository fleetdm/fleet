package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261007155733, Down_20261007155733)
}

func Up_20261007155733(tx *sql.Tx) error {
	const batchSize = 1000

	var maxInstallID sql.NullInt64
	err := tx.QueryRow(`SELECT MAX(id) FROM host_in_house_software_installs`).Scan(&maxInstallID)
	if err != nil {
		return fmt.Errorf("selecting max host_in_house_software_installs id: %w", err)
	}

	// Batch by id range, MySQL does not allow LIMIT on an UPDATE with a JOIN
	for startID := int64(0); startID < maxInstallID.Int64; startID += batchSize {
		// Use the install command uuid rather than the verification command uuid, verification_command_uuid is null on rows affected by the bug
		_, err = tx.Exec(`
UPDATE host_in_house_software_installs hihsi
JOIN nano_command_results ncr ON ncr.command_uuid = hihsi.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE hihsi.verification_at IS NULL
	AND hihsi.verification_failed_at IS NULL
	AND hihsi.canceled = 0
	AND ncr.status IN ('Error', 'CommandFormatError')
	AND hihsi.id > ? AND hihsi.id <= ?`, startID, startID+batchSize)
		if err != nil {
			return fmt.Errorf("set verification_failed_at on in-house app installs that errored after id %d: %w", startID, err)
		}
	}
	return nil
}

func Down_20261007155733(tx *sql.Tx) error {
	return nil
}
