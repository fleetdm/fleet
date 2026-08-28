package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260828101500, Down_20260828101500)
}

// Refetch requests used to be accepted for Android hosts and set this flag, but
// nothing ever clears it for them: it is only reset by the osquery detail path
// and by the Apple MDM DeviceInformation response, and the Android host insert
// leaves the column at its default. Hosts flagged before the request was
// rejected would keep it set forever.
func Up_20260828101500(tx *sql.Tx) error {
	// updated_at is ON UPDATE CURRENT_TIMESTAMP and is assigned to itself so that
	// clearing the flag doesn't make every Android host look freshly updated.
	if _, err := tx.Exec(`
		UPDATE hosts
		SET refetch_requested = 0, updated_at = updated_at
		WHERE platform = 'android' AND refetch_requested = 1
	`); err != nil {
		return fmt.Errorf("clearing refetch_requested on android hosts: %w", err)
	}

	return nil
}

func Down_20260828101500(tx *sql.Tx) error {
	return nil
}
