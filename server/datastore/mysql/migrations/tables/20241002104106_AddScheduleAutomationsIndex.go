package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241002104106, Down_20241002104106)
}

func Up_20241002104106(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE queries
		ADD COLUMN is_scheduled BOOLEAN GENERATED ALWAYS AS (schedule_interval > 0) STORED NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("error creating generated column is_scheduled: %w", err)
	}

	_, err = tx.Exec(`
		CREATE INDEX idx_queries_schedule_automations ON queries (is_scheduled, automations_enabled)
	`)
	if err != nil {
		return fmt.Errorf("error creating index idx_queries_schedule_automations: %w", err)
	}

	return nil
}

func Down_20241002104106(tx *sql.Tx) error {
	return nil
}
