package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240925171917, Down_20240925171917)
}

func Up_20240925171917(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE INDEX idx_queries_schedule_automations ON queries (schedule_interval, automations_enabled)
	`)
	if err != nil {
		return fmt.Errorf("error creating index idx_queries_schedule_automations: %w", err)
	}

	return nil
}

func Down_20240925171917(tx *sql.Tx) error {
	return nil
}
