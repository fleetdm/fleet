package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250121094045, Down_20250121094045)
}

func Up_20250121094045(tx *sql.Tx) error {
	stmt := `
CREATE TABLE IF NOT EXISTS host_disk_encryption_keys_archive (
  -- Since we may never delete rows from this table, we use a large PRIMARY KEY
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  host_id int unsigned NOT NULL,
  base64_encrypted text COLLATE utf8mb4_unicode_ci NOT NULL,
  base64_encrypted_salt varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  key_slot tinyint unsigned DEFAULT NULL,
  decryptable tinyint(1) DEFAULT NULL,
  original_created_at TIMESTAMP(6) NOT NULL,
  original_updated_at TIMESTAMP(6) NULL,
  reset_requested tinyint(1) NOT NULL DEFAULT '0',
  client_error varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  created_at TIMESTAMP(6) NOT NULL DEFAULT NOW(6),
  KEY idx_host_disk_encryption_keys_archive_host_created_at (host_id, created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to create host_disk_encryption_keys_archive: %w", err)
	}

	return nil
}

func Down_20250121094045(_ *sql.Tx) error {
	return nil
}
