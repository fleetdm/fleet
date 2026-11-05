package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20261104165437, Down_20261104165437)
}

func Up_20261104165437(tx *sql.Tx) error {
	// Replace the single-column user_id index with a composite (user_id,
	// accessed_at) index so that the MAX(accessed_at) subquery in
	// userLastActivitySelect (used on every authenticated request) is
	// resolved by the index alone ("Select tables optimized away") instead
	// of scanning all of the user's session rows.
	// See https://github.com/fleetdm/fleet/issues/55579.

	// Create the composite index first so the old one remains as a
	// fallback if this step fails or is interrupted.
	if !indexExistsTx(tx, "sessions", "idx_sessions_user_id_accessed_at") {
		if _, err := tx.Exec(`ALTER TABLE sessions ADD INDEX idx_sessions_user_id_accessed_at (user_id, accessed_at)`); err != nil {
			return err
		}
	}

	if indexExistsTx(tx, "sessions", "idx_sessions_user_id") {
		if _, err := tx.Exec(`ALTER TABLE sessions DROP INDEX idx_sessions_user_id`); err != nil {
			return err
		}
	}

	return nil
}

func Down_20261104165437(tx *sql.Tx) error {
	return nil
}
