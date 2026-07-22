package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260722203845, Down_20260722203845)
}

func Up_20260722203845(tx *sql.Tx) error {
	// Track the last-processed Google Pub/Sub message per Android device so the
	// AMAPI notification handler can deduplicate at-least-once redeliveries
	// (same messageId) and drop out-of-order deliveries (older event timestamp).
	if _, err := tx.Exec(`ALTER TABLE android_devices
		ADD COLUMN last_pubsub_message_id VARCHAR(255) DEFAULT NULL,
		ADD COLUMN last_pubsub_event_time TIMESTAMP(6) NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("add pubsub dedup columns to android_devices: %w", err)
	}

	return nil
}

func Down_20260722203845(tx *sql.Tx) error {
	return nil
}
