package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240205121956, Down_20240205121956)
}

func Up_20240205121956(tx *sql.Tx) error {
	stmt := `
		CREATE TABLE host_mdm_actions (
			host_id INT UNSIGNED NOT NULL,
			lock_ref VARCHAR(36) NULL,
			wipe_ref VARCHAR(36) NULL,
			suspended TINYINT(1) NOT NULL DEFAULT FALSE,
			PRIMARY KEY (host_id)
		)
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("create table host_mdm_actions: %w", err)
	}

	return nil
}

func Down_20240205121956(tx *sql.Tx) error {
	return nil
}
