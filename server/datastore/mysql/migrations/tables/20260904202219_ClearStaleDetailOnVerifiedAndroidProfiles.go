package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260904202219, Down_20260904202219)
}

// When an Android profile moved to verified, the detail recorded by an earlier
// failed report was carried into the upsert, so a profile that had recovered
// still displayed its old failure message. The write path no longer does that,
// but rows already in this state are only rewritten when the profile changes
// again, which may never happen. Clear them here.
//
// Scoped to verified rows: a failed or pending profile's detail is its current
// error message and has to survive.
func Up_20260904202219(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		UPDATE host_mdm_android_profiles
		SET detail = '', updated_at = updated_at
		WHERE status = 'verified' AND detail != ''
	`); err != nil {
		return fmt.Errorf("clearing stale detail on verified Android profiles: %w", err)
	}

	return nil
}

func Down_20260904202219(tx *sql.Tx) error {
	return nil
}
