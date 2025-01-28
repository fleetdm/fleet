package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240707134035, Down_20240707134035)
}

func Up_20240707134035(tx *sql.Tx) error {
	// UUID is a 36-character string with the most common 8-4-4-4-12 format, xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// Reference: https://en.wikipedia.org/wiki/Universally_unique_identifier#Textual_representation
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD COLUMN uuid VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL`); err != nil {
		return fmt.Errorf("failed to add `uuid` column to `calendar_events` table: %w", err)
	}

	// Generate UUIDs for existing calendar events, without changing the updated_at timestamp
	if _, err := tx.Exec(`UPDATE calendar_events SET uuid = UUID(), updated_at = updated_at`); err != nil {
		return fmt.Errorf("failed to generate UUIDs for existing calendar events: %w", err)
	}

	// Add unique constraint to uuid column
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD CONSTRAINT idx_calendar_events_uuid_unique UNIQUE (uuid)`); err != nil {
		return fmt.Errorf("failed to add unique constraint to `uuid` column in `calendar_events` table: %w", err)
	}

	return nil
}

func Down_20240707134035(_ *sql.Tx) error {
	return nil
}
