package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250121094045, Down_20250121094045)
}

func Up_20250121094045(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_disk_encryption_keys
		MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT NOW(6),
		MODIFY COLUMN updated_at TIMESTAMP(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6)`)
	if err != nil {
		return fmt.Errorf("failed to alter host_disk_encryption_keys table: %w", err)
	}

	stmt := `
CREATE TABLE IF NOT EXISTS host_disk_encryption_keys_archive (
  -- Since we may never delete rows from this table, we use a large PRIMARY KEY
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  host_id int unsigned NOT NULL,
  hardware_serial VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  base64_encrypted text COLLATE utf8mb4_unicode_ci NOT NULL,
  base64_encrypted_salt varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  key_slot tinyint unsigned DEFAULT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT NOW(6),
  KEY idx_host_disk_encryption_keys_archive_host_created_at (host_id, created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to create host_disk_encryption_keys_archive: %w", err)
	}

	// Copy all existing rows from host_disk_encryption_keys to host_disk_encryption_keys_archive
	const copyKeysToArchiveQuery = `
INSERT INTO host_disk_encryption_keys_archive (host_id, base64_encrypted, base64_encrypted_salt, key_slot, created_at)
SELECT host_id, base64_encrypted, base64_encrypted_salt, key_slot, created_at
FROM host_disk_encryption_keys`
	_, err = tx.Exec(copyKeysToArchiveQuery)
	if err != nil {
		return fmt.Errorf("failed to copy existing rows to host_disk_encryption_keys_archive: %w", err)
	}

	// Update the hardware_serial column to match the host table
	const updateHardwareSerialQuery = `
UPDATE host_disk_encryption_keys_archive
JOIN hosts ON host_disk_encryption_keys_archive.host_id = hosts.id
SET host_disk_encryption_keys_archive.hardware_serial = hosts.hardware_serial`
	_, err = tx.Exec(updateHardwareSerialQuery)
	if err != nil {
		return fmt.Errorf("failed to update host_disk_encryption_keys_archive.hardware_serial: %w", err)
	}

	return nil
}

func Down_20250121094045(_ *sql.Tx) error {
	return nil
}
