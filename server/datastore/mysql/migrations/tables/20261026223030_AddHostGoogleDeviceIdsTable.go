package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261026223030, Down_20261026223030)
}

// Up_20261026223030 adds the table that binds iOS, iPadOS and Android hosts to their device record in
// Google. It is not a column on host_conditional_access because those rows are deleted when a bypass is
// consumed or bypasses are turned off.
func Up_20261026223030(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_google_device_ids (
	host_id          INT UNSIGNED NOT NULL,
	google_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	created_at       TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_id),
	UNIQUE KEY idx_host_google_device_ids_google_device_id (google_device_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_google_device_ids table: %w", err)
	}
	return nil
}

func Down_20261026223030(tx *sql.Tx) error {
	return nil
}
