package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261022203155, Down_20261022203155)
}

func Up_20261022203155(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE certificate_authorities
		MODIFY COLUMN type
			ENUM('digicert','ndes_scep_proxy','custom_scep_proxy','hydrant','smallstep','custom_est_proxy','ejbca','google_cas')
			CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL`); err != nil {
		return fmt.Errorf("adding google_cas to certificate_authorities.type enum: %w", err)
	}

	if !columnExists(tx, "certificate_authorities", "google_cas_project") {
		if _, err := tx.Exec(`
			ALTER TABLE certificate_authorities
			ADD COLUMN google_cas_project VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER ejbca_username_template,
			ADD COLUMN google_cas_location VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER google_cas_project,
			ADD COLUMN google_cas_ca_pool VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER google_cas_location,
			ADD COLUMN google_cas_api_key_json_encrypted BLOB NULL AFTER google_cas_ca_pool`); err != nil {
			return fmt.Errorf("adding Google CAS columns to certificate_authorities: %w", err)
		}
	}
	return nil
}

func Down_20261022203155(tx *sql.Tx) error {
	return nil
}
