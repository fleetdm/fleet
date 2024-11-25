package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241121111624, Down_20241121111624)
}

func Up_20241121111624(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE pki_certificates (
    name VARCHAR(255) PRIMARY KEY,
    cert_pem BLOB NULL, -- 65,535 bytes max
    key_pem BLOB NOT NULL, -- 65,535 bytes max
    not_valid_after DATETIME NULL,
    sha256 BINARY(32) NULL,
    sha256_hex CHAR(64) GENERATED ALWAYS AS (LOWER(HEX(sha256))) VIRTUAL NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create pki_certificates table: %w", err)
	}
	return nil
}

func Down_20241121111624(_ *sql.Tx) error {
	return nil
}
