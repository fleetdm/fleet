package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250603113055, Down_20250603113055)
}

func Up_20250603113055(tx *sql.Tx) error {
	// host_lifecycle_events table:
	// host_serial
	// host_uuid
	// host_id
	// event_type
	// created_at
	// activity_id
	createStmt := `
	CREATE TABLE IF NOT EXISTS host_lifecycle_events (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		host_serial varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
		host_uuid VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		host_id INT UNSIGNED NOT NULL,
		event_type ENUM('started_mdm_setup', 'completed_mdm_setup', 'started_mdm_migration', 'completed_mdm_migration') COLLATE utf8mb4_unicode_ci NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		activity_id INT UNSIGNED,
		INDEX idx_host_lifecycle_events_host_id(host_id),
		INDEX idx_host_lifecycle_events_host_uuid(host_uuid),
		INDEX idx_host_lifecycle_events_event_type(event_type)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`
	_, err := tx.Exec(createStmt)
	if err != nil {
		return fmt.Errorf("creating host_lifecycle_events table: %w", err)
	}

	return nil
}

func Down_20250603113055(tx *sql.Tx) error {
	return nil
}
