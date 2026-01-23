package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260119220029, Down_20260119220029)
}

func Up_20260119220029(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_conditional_access (
			id int unsigned NOT NULL AUTO_INCREMENT,
			host_id int unsigned NOT NULL,
			bypassed_at timestamp,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_host_conditional_access_host_id (host_id)
		)
	`); err != nil {
		return fmt.Errorf("creating host_conditional_access table: %w", err)
	}

	return nil
}

func Down_20260119220029(tx *sql.Tx) error {
	return nil
}
