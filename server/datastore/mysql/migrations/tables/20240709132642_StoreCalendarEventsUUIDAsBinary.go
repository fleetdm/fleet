package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240709132642, Down_20240709132642)
}

func Up_20240709132642(tx *sql.Tx) error {
	// Implementation based on: https://dev.mysql.com/blog-archive/storing-uuid-values-in-mysql-tables/

	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD COLUMN uuid_bin BINARY(16) NOT NULL`); err != nil {
		return fmt.Errorf("failed to add `uuid_bin` column to `calendar_events` table: %w", err)
	}

	// Convert existing UUIDs to binary format
	if _, err := tx.Exec(`UPDATE calendar_events SET uuid_bin = UNHEX(REPLACE(uuid,'-','')), updated_at = updated_at`); err != nil {
		return fmt.Errorf("failed to convert UUIDs to binary form for existing calendar events: %w", err)
	}

	// Add unique constraint to uuid_bin column
	if _, err := tx.Exec(`ALTER TABLE calendar_events ADD CONSTRAINT idx_calendar_events_uuid_bin_unique UNIQUE (uuid_bin)`); err != nil {
		return fmt.Errorf("failed to add unique constraint to `uuid_bin` column in `calendar_events` table: %w", err)
	}

	// Drop existing uuid column
	if _, err := tx.Exec(`ALTER TABLE calendar_events DROP COLUMN uuid`); err != nil {
		return fmt.Errorf("failed to drop `uuid` column from `calendar_events` table: %w", err)
	}

	// Add a new GENERATED uuid column
	if _, err := tx.Exec(
		`ALTER TABLE calendar_events ADD COLUMN uuid VARCHAR(36) COLLATE utf8mb4_unicode_ci GENERATED ALWAYS AS (
			(INSERT(
			    INSERT(
				  INSERT(
					INSERT(hex(uuid_bin),9,0,'-'),
					14,0,'-'),
				  19,0,'-'),
				24,0,'-')
			 )) VIRTUAL`); err != nil {
		return fmt.Errorf("failed to add `uuid` column to `calendar_events` table: %w", err)
	}

	return nil
}

func Down_20240709132642(_ *sql.Tx) error {
	return nil
}
