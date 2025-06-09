package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250609102714, Down_20250609102714)
}

func Up_20250609102714(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_certificate_sources  (
	id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	host_certificate_id  BIGINT UNSIGNED NOT NULL,
	source               ENUM('system', 'user') NOT NULL,
	username             VARCHAR(255) NOT NULL,
	created_at           DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

	UNIQUE KEY idx_host_certificate_sources_unique (host_certificate_id, source, username),
	CONSTRAINT fk_host_certificate_sources_host_certificate_id
		FOREIGN KEY (host_certificate_id) REFERENCES host_certificates (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)

	if err != nil {
		return fmt.Errorf("failed to create host_certificate_sources table: %w", err)
	}

	// all existing host_certificates entry now need to have a corresponding
	// entry in host_certificate_sources with the source = 'system'. Adding even
	// if the certificate is soft-deleted, so that we know what source it was
	// from.
	_, err = tx.Exec(`
INSERT INTO host_certificate_sources
	(host_certificate_id, source, username)
	SELECT
		id,
		'system',
		''
	FROM
		host_certificates
`)
	if err != nil {
		return fmt.Errorf("failed to insert into host_certificate_sources: %w", err)
	}
	return nil
}

func Down_20250609102714(tx *sql.Tx) error {
	return nil
}
