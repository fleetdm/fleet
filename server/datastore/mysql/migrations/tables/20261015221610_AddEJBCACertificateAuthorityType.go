package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261015221610, Down_20261015221610)
}

func Up_20261015221610(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE certificate_authorities
		MODIFY COLUMN type
			ENUM('digicert','ndes_scep_proxy','custom_scep_proxy','hydrant','smallstep','custom_est_proxy','ejbca')
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL`); err != nil {
		return fmt.Errorf("adding ejbca to certificate_authorities.type enum: %w", err)
	}

	if !columnExists(tx, "certificate_authorities", "mtls_cert_pem") {
		if _, err := tx.Exec(`
			ALTER TABLE certificate_authorities
			ADD COLUMN mtls_cert_pem TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER client_secret_encrypted,
			ADD COLUMN mtls_key_encrypted BLOB NULL AFTER mtls_cert_pem,
			ADD COLUMN trust_ca_bundle_pem TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER mtls_key_encrypted,
			ADD COLUMN ejbca_ca_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER trust_ca_bundle_pem,
			ADD COLUMN ejbca_certificate_profile_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
				AFTER ejbca_ca_name,
			ADD COLUMN ejbca_end_entity_profile_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
				AFTER ejbca_certificate_profile_name,
			ADD COLUMN ejbca_username_template VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
				AFTER ejbca_end_entity_profile_name`); err != nil {
			return fmt.Errorf("adding EJBCA columns to certificate_authorities: %w", err)
		}
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_mdm_managed_certificates
		MODIFY COLUMN type ENUM('digicert','custom_scep_proxy','ndes','smallstep','ejbca')
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
		ALGORITHM=INSTANT`); err != nil {
		return fmt.Errorf("adding ejbca to host_mdm_managed_certificates.type enum: %w", err)
	}

	// Profiles are linked to their CA through these prefix variables. A constant time keeps the generated schema deterministic.
	stmt, args, err := sqlx.Named(`
		INSERT IGNORE INTO fleet_variables (name, is_prefix, created_at)
		VALUES ('FLEET_VAR_EJBCA_DATA_', 1, :created_at), ('FLEET_VAR_EJBCA_PASSWORD_', 1, :created_at)`,
		map[string]any{"created_at": time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return fmt.Errorf("preparing insert of EJBCA fleet variables: %w", err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("inserting EJBCA fleet variables: %w", err)
	}
	return nil
}

func Down_20261015221610(tx *sql.Tx) error {
	return nil
}
