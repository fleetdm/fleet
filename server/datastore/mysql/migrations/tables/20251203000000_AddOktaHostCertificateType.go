package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251203000000, Down_20251203000000)
}

func Up_20251203000000(tx *sql.Tx) error {
	// Add 'okta' to host_mdm_managed_certificates.type enum
	_, err := tx.Exec(`
ALTER TABLE host_mdm_managed_certificates
MODIFY COLUMN type ENUM('digicert','custom_scep_proxy','ndes','smallstep','okta') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ndes';`)
	if err != nil {
		return fmt.Errorf("failed to modify host_mdm_managed_certificates table: %w", err)
	}

	// Add 'okta' to certificate_authorities.type enum
	// Note: Okta shares username, password_encrypted, and challenge_url fields with Smallstep
	_, err = tx.Exec(`
ALTER TABLE certificate_authorities
MODIFY COLUMN type ENUM('digicert', 'ndes_scep_proxy', 'custom_scep_proxy', 'hydrant', 'smallstep', 'custom_est_proxy', 'okta') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;`)
	if err != nil {
		return fmt.Errorf("failed to modify certificate_authorities table: %w", err)
	}

	return nil
}

func Down_20251203000000(tx *sql.Tx) error {
	return nil
}
