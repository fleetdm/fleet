package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250211141712, Down_20250211141712)
}

func Up_20250211141712(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_certificates (
	cert_id             		BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	host_id         			INT UNSIGNED NOT NULL,
	not_valid_after				DATETIME(6) NOT NULL,
	not_valid_before 			DATETIME(6) NOT NULL,
	certificate_authority 		TINYINT(1) NOT NULL,
	common_name 				VARCHAR(255) NOT NULL,
	key_algorithm 				VARCHAR(255) NOT NULL,
	key_strength 				INT NOT NULL,
	key_usage 					VARCHAR(255) NOT NULL,
	serial 						VARCHAR(255) NOT NULL,
	signing_algorithm 			VARCHAR(255) NOT NULL,
	subject_country 			VARCHAR(2) NOT NULL,
	subject_org			 		VARCHAR(255) NOT NULL,
	subject_org_unit 			VARCHAR(255) NOT NULL,
	subject_common_name 		VARCHAR(255) NOT NULL,
	issuer_country 				VARCHAR(2) NOT NULL,
	issuer_org					VARCHAR(255) NOT NULL,
	issuer_org_unit 			VARCHAR(255) NOT NULL,
	issuer_common_name 			VARCHAR(255) NOT NULL,
	checksum 					BINARY(20) NOT NULL,

	PRIMARY KEY (cert_id),
	UNIQUE KEY idx_host_id_cert_id (host_id, cert_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)

	return err
}

func Down_20250211141712(tx *sql.Tx) error {
	return nil
}
