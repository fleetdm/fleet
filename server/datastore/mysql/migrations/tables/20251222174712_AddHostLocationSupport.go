package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251222174712, Down_20251222174712)
}

func Up_20251222174712(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE host_locations (
		host_id INT UNSIGNED NOT NULL,
		latitude DECIMAL(10, 8),
		longitude DECIMAL(11, 8),
		created_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
		updated_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

		PRIMARY KEY (host_id),
		FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
	) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table host_locations: %w", err)
	}
	return nil
}

func Down_20251222174712(tx *sql.Tx) error {
	return nil
}
