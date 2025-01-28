package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240710155623, Down_20240710155623)
}

func Up_20240710155623(tx *sql.Tx) error {
	// `hosts.last_enrolled_at` contains the date the osquery agent enrolled.
	//
	// A bug in v4.51.0 caused the `last_enrolled_at` column to be set to
	// '2000-01-01 00:00:00' (aka the "Never" date) when macOS hosts perform a
	// MDM re-enrollment (see https://github.com/fleetdm/fleet/issues/20059
	// for more details).
	//
	// We cannot restore the exact date of the osquery enrollment but
	// `host_disks.created_at` is a good approximation.
	if _, err := tx.Exec(`
		UPDATE hosts h
		JOIN host_disks hd ON h.id=hd.host_id
		SET h.last_enrolled_at = hd.created_at, h.updated_at = h.updated_at
		WHERE h.platform = 'darwin' AND h.last_enrolled_at = '2000-01-01 00:00:00';`,
	); err != nil {
		return fmt.Errorf("failed to update hosts.last_enrolled_at: %w", err)
	}

	return nil
}

func Down_20240710155623(tx *sql.Tx) error {
	return nil
}
