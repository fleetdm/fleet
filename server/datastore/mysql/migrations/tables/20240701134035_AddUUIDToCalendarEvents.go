package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240701134035, Down_20240701134035)
}

func Up_20240701134035(tx *sql.Tx) error {
	// UUID is a 36-character string with the most common 8-4-4-4-12 format, xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// Reference: https://en.wikipedia.org/wiki/Universally_unique_identifier#Textual_representation
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD COLUMN uuid VARCHAR(36) NOT NULL`); err != nil {
		return fmt.Errorf("failed to add `uuid` column to `calendar_events` table: %w", err)
	}

	// Generate UUIDs for existing calendar events
	if _, err := tx.Exec(`UPDATE calendar_events SET uuid = UUID()`); err != nil {
		return fmt.Errorf("failed to generate UUIDs for existing calendar events: %w", err)
	}

	// Add unique constraint to uuid column
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD CONSTRAINT idx_calendar_events_uuid_unique UNIQUE (uuid)`); err != nil {
		return fmt.Errorf("failed to add unique constraint to `uuid` column in `calendar_events` table: %w", err)
	}

	return nil
}

func Down_20240701134035(_ *sql.Tx) error {
	return nil
}
