package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230629140529, Down_20230629140529)
}

func Up_20230629140529(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE wstep_serials (
	serial         BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
	created_at     timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,

	PRIMARY KEY (serial)
);`)
	if err != nil {
		return err
	}

	// assume that the first serial number is assigned to the CA cert
	_, err = tx.Exec(`
ALTER TABLE wstep_serials AUTO_INCREMENT = 2;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE wstep_certificates (
	serial             BIGINT(20) UNSIGNED NOT NULL,
	name               VARCHAR(1024) NOT NULL,
	not_valid_before   DATETIME NOT NULL,
	not_valid_after    DATETIME NOT NULL,
	certificate_pem    TEXT NOT NULL,
	revoked 		   TINYINT(1) NOT NULL DEFAULT 0,
	created_at         timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at         timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (serial),
	FOREIGN KEY (serial) REFERENCES wstep_serials (serial)
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE wstep_cert_auth_associations (
	id             VARCHAR(255) NOT NULL,
	sha256		   CHAR(64) NOT NULL,
	created_at     timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at     timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id, sha256)
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return err
	}

	return nil
}

func Down_20230629140529(tx *sql.Tx) error {
	return nil
}
