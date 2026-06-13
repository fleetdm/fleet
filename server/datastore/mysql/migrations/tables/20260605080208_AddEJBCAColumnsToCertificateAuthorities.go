package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260605080208, Down_20260605080208)
}

// Up_20260605080208 extends the certificate_authorities table to support EJBCA
// REST integration. Adds the mTLS material columns (client cert + encrypted key
// + optional trust bundle) plus the four EJBCA-side identifier columns the API
// requires on each pkcs10enroll call. The `type` ENUM is extended with 'ejbca'.
//
// No enrollment-code column: Fleet generates the per-issuance EJBCA `password`
// internally per enrollment and never persists it (see openspec
// add-ejbca-rest-ca-poc REQ-CA-EJBCA-7).
//
// certificate_user_principal_names already exists for DigiCert and is reused
// for EJBCA's UPN SAN templating.
func Up_20260605080208(tx *sql.Tx) error {
	stmt := `
ALTER TABLE certificate_authorities
	MODIFY COLUMN type ENUM(
		'digicert',
		'ndes_scep_proxy',
		'custom_scep_proxy',
		'hydrant',
		'custom_est_proxy',
		'smallstep',
		'ejbca'
	) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	ADD COLUMN client_cert_pem BLOB,
	ADD COLUMN client_key_encrypted BLOB,
	ADD COLUMN trust_ca_bundle_pem BLOB,
	ADD COLUMN ejbca_ca_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	ADD COLUMN ejbca_certificate_profile VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	ADD COLUMN ejbca_end_entity_profile VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	ADD COLUMN ejbca_username_template VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("alter certificate_authorities for EJBCA support: %w", err)
	}
	return nil
}

func Down_20260605080208(tx *sql.Tx) error {
	return nil
}
