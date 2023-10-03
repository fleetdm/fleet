package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230927111208, Down_20230927111208)
}

func Up_20230927111208(tx *sql.Tx) error {
	sql := `
ALTER TABLE 
    host_script_results 
ADD INDEX 
    idx_host_script_created_at (host_id, script_id, created_at);
	`
	if _, err := tx.Exec(sql); err != nil {
		return fmt.Errorf("add index host_script_created_at to host_script_results: %w", err)
	}

	return nil
}

func Down_20230927111208(tx *sql.Tx) error {
	return nil
}
