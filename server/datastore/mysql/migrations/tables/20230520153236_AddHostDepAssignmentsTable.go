package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230520153236, Down_20230520153236)
}

func Up_20230520153236(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE host_dep_assignments (
            host_id INT(10) UNSIGNED NOT NULL,
            added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP NULL,

           PRIMARY KEY (host_id)
        )`)
	if err != nil {
		return fmt.Errorf("failed to create host_dep_assignments table: %w", err)
	}

	return nil
}

func Down_20230520153236(tx *sql.Tx) error {
	return nil
}
