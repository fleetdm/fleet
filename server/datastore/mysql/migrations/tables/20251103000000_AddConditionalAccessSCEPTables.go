package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251103000000, Down_20251103000000)
}

func Up_20251103000000(tx *sql.Tx) error {
	// Create conditional_access_scep_serials table first (referenced by foreign key)
	// Reserve serial number 1 for system use, similar to host identity SCEP
	_, err := tx.Exec(`
		CREATE TABLE conditional_access_scep_serials (
			serial bigint unsigned NOT NULL AUTO_INCREMENT,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			PRIMARY KEY (serial)
		) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to create conditional_access_scep_serials table: %w", err)
	}

	// Create conditional_access_scep_certificates table
	_, err = tx.Exec(`
		CREATE TABLE conditional_access_scep_certificates (
			serial bigint unsigned NOT NULL,
			host_id int unsigned NOT NULL,
			name varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
			not_valid_before datetime NOT NULL,
			not_valid_after datetime NOT NULL,
			certificate_pem text COLLATE utf8mb4_unicode_ci NOT NULL,
			revoked tinyint(1) NOT NULL DEFAULT '0',
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			PRIMARY KEY (serial),
			KEY idx_conditional_access_host_id (host_id),
			CONSTRAINT conditional_access_scep_certificates_ibfk_1 FOREIGN KEY (serial) REFERENCES conditional_access_scep_serials (serial),
			CONSTRAINT conditional_access_scep_certificates_chk_1 CHECK ((substr(certificate_pem,1,27) = _utf8mb4'-----BEGIN CERTIFICATE-----'))
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("failed to create conditional_access_scep_certificates table: %w", err)
	}

	return nil
}

func Down_20251103000000(_ *sql.Tx) error {
	return nil
}
