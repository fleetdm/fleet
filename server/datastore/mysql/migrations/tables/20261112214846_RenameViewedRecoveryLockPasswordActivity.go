package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261112214846, Down_20261112214846)
}

// The activity_type index keeps the UPDATE selective, and batching would not help because the migration runs in a single transaction.
func Up_20261112214846(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`UPDATE activity_past SET activity_type = 'read_host_recovery_lock_password' WHERE activity_type = 'viewed_host_recovery_lock_password'`,
	); err != nil {
		return fmt.Errorf("renaming viewed_host_recovery_lock_password activities: %w", err)
	}
	return nil
}

func Down_20261112214846(tx *sql.Tx) error {
	return nil
}
