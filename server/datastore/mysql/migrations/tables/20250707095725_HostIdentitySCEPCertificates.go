package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250707095725, Down_20250707095725)
}

func Up_20250707095725(tx *sql.Tx) error {
	// Create host_identity_scep_serials table first (referenced by foreign key)
	// In SCEP (Simple Certificate Enrollment Protocol) implementations, it's common practice to reserve serial number 1 for the CA (Certificate Authority) certificate itself or for other system-level certificates.
	_, err := tx.Exec(`
		CREATE TABLE host_identity_scep_serials (
			serial bigint unsigned NOT NULL AUTO_INCREMENT,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			PRIMARY KEY (serial)
		) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to create host_identity_scep_serials table: %w", err)
	}

	// Create host_identity_scep_certificates table
	_, err = tx.Exec(`
		CREATE TABLE host_identity_scep_certificates (
			serial bigint unsigned NOT NULL,
			host_id int unsigned NULL,
			name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			not_valid_before datetime NOT NULL,
			not_valid_after datetime NOT NULL,
			certificate_pem text COLLATE utf8mb4_unicode_ci NOT NULL, -- 65K max, stored for debug/auditing but not used
			public_key_raw VARBINARY(100) NOT NULL, -- for quick retrieval/verification
			revoked tinyint(1) NOT NULL DEFAULT '0',
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			PRIMARY KEY (serial),
			KEY idx_host_id_scep_name (name), -- for quick revocation
			KEY idx_host_id_scep_host_id (host_id),
			CONSTRAINT host_identity_scep_certificates_ibfk_1 FOREIGN KEY (serial) REFERENCES host_identity_scep_serials (serial),
			CONSTRAINT host_identity_scep_certificates_chk_1 CHECK ((substr(certificate_pem,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----'))
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to create host_identity_scep_certificates table: %w", err)
	}

	return nil
}

func Down_20250707095725(tx *sql.Tx) error {
	// Drop tables in reverse order due to foreign key constraint
	_, err := tx.Exec(`DROP TABLE IF EXISTS host_identity_scep_certificates`)
	if err != nil {
		return fmt.Errorf("failed to drop host_identity_scep_certificates table: %w", err)
	}

	_, err = tx.Exec(`DROP TABLE IF EXISTS host_identity_scep_serials`)
	if err != nil {
		return fmt.Errorf("failed to drop host_identity_scep_serials table: %w", err)
	}

	return nil
}
