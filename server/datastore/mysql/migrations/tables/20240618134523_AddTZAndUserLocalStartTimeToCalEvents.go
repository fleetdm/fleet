package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240618134523, Down_20240618134523)
}

func Up_20240618134523(tx *sql.Tx) error {
	// timezone an IANA timezone name, e.g. "America/Argentina/Buenos_Aires"
	// user_local_time includes UTC offset, e.g. "2024-06-18T13:27:18−07:00 UTC−07:00"
	// both can be NULL since these fields will be written only once the next cron runs after migration
	if _, err := tx.Exec(`
	ALTER TABLE
		calendar_events
	ADD COLUMN
		timezone TEXT NULL,
	ADD COLUMN
		user_local_start_time TEXT NULL
	`); err != nil {
		return fmt.Errorf("failed to add `timezone` and/or `user_local_start_time` columns to `calendar_events` table: %w", err)
	}
	return nil
}

func Down_20240618134523(tx *sql.Tx) error {
	return nil
}
