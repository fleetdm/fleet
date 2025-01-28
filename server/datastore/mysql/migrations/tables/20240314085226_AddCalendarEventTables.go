package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240314085226, Down_20240314085226)
}

func Up_20240314085226(tx *sql.Tx) error {
	// TODO(lucas): Check if we need more indexes.

	if _, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS calendar_events (
		id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  		email VARCHAR(255) NOT NULL,
		start_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		end_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		event JSON NOT NULL,

		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

		UNIQUE KEY idx_one_calendar_event_per_email (email)
	);
`); err != nil {
		return fmt.Errorf("create calendar_events table: %w", err)
	}

	if _, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS host_calendar_events (
		id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		host_id INT(10) UNSIGNED NOT NULL,
		calendar_event_id INT(10) UNSIGNED NOT NULL,
		webhook_status TINYINT NOT NULL,

		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

		UNIQUE KEY idx_one_calendar_event_per_host (host_id),
		FOREIGN KEY (calendar_event_id) REFERENCES calendar_events(id) ON DELETE CASCADE
	);
`); err != nil {
		return fmt.Errorf("create host_calendar_events table: %w", err)
	}

	return nil
}

func Down_20240314085226(tx *sql.Tx) error {
	return nil
}
