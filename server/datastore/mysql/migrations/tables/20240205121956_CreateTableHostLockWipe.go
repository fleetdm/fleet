package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240205121956, Down_20240205121956)
}

func Up_20240205121956(tx *sql.Tx) error {
	// Adding a new table for this data as the existing `host_mdm` table is related more closely to
	// enrollment logic.
	// lock_ref and wipe_ref are the UUIDs of the actions taken to lock or wipe a host. These could
	// point at MDM commands or script executions, depending on the host platform. suspended
	// indicates whether or not further actions on this host are suspended (will be set to true
	// while the wipe or lock action is pending, and set to false again once the action has completed).
	stmt := `
		CREATE TABLE host_mdm_actions (
			host_id INT UNSIGNED NOT NULL,
			lock_ref VARCHAR(36) NULL,
			wipe_ref VARCHAR(36) NULL,
			suspended TINYINT(1) NOT NULL DEFAULT FALSE,
			PRIMARY KEY (host_id)
		) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("create table host_mdm_actions: %w", err)
	}

	return nil
}

func Down_20240205121956(tx *sql.Tx) error {
	return nil
}
