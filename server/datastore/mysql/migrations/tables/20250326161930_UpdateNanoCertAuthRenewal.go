package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250326161930, Down_20250326161930)
}

func Up_20250326161930(tx *sql.Tx) error {
	_, err := tx.Exec(`
UPDATE
	nano_cert_auth_associations ncaa
	JOIN nano_command_results ncr ON renew_command_uuid = command_uuid
		AND ncaa.id = ncr.id
	SET
		renew_command_uuid = NULL
WHERE
	ncr.status = 'Acknowledged'`)
	if err != nil {
		return fmt.Errorf("failed to update nano_cert_auth_associations: %w", err)
	}

	return nil
}

func Down_20250326161930(tx *sql.Tx) error {
	return nil
}
