package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260102145839, Down_20260102145839)
}

func Up_20260102145839(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE host_last_known_locations (
		host_id INT UNSIGNED NOT NULL,
		latitude DECIMAL(10, 8),
		longitude DECIMAL(11, 8),
		created_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
		updated_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

		PRIMARY KEY (host_id)
	) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table host_last_known_locations: %w", err)
	}
	return nil
}

func Down_20260102145839(tx *sql.Tx) error {
	return nil
}
