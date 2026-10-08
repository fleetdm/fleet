package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261008230604, Down_20261008230604)
}

func Up_20261008230604(tx *sql.Tx) error {
	// Kept out of the software table so that metadata which isn't part of software identity (and
	// therefore not of its checksum) can be collected without creating new software rows.
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS software_metadata (
  software_id BIGINT UNSIGNED NOT NULL,
  epoch       INT UNSIGNED DEFAULT NULL,
  PRIMARY KEY (software_id),
  CONSTRAINT fk_software_metadata_software_id FOREIGN KEY (software_id) REFERENCES software (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create software_metadata table: %w", err)
	}
	return nil
}

func Down_20261008230604(tx *sql.Tx) error {
	return nil
}
