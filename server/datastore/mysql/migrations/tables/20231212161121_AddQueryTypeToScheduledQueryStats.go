package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231212161121, Down_20231212161121)
}

func Up_20231212161121(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE scheduled_query_stats
		ADD COLUMN query_type TINYINT NOT NULL DEFAULT 0;
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add query_type to scheduled_query_stats: %w", err)
	}

	// Add query_type to primary key
	stmt = `
		ALTER TABLE scheduled_query_stats
		DROP PRIMARY KEY,
		ADD PRIMARY KEY (host_id, scheduled_query_id, query_type);
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add query_type to scheduled_query_stats primary key: %w", err)
	}

	return nil
}

func Down_20231212161121(*sql.Tx) error {
	/*
		ALTER TABLE scheduled_query_stats
		DROP PRIMARY KEY,
		ADD PRIMARY KEY (host_id, scheduled_query_id);
		ALTER TABLE scheduled_query_stats
		DROP COLUMN query_type;
	*/
	return nil
}
