package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250923120000, Down_20250923120000)
}

func Up_20250923120000(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE host_mdm_managed_certificates 
MODIFY COLUMN type ENUM('digicert','custom_scep_proxy','ndes','smallstep') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'ndes';	`)
	if err != nil {
		return fmt.Errorf("failed to modify host_mdm_managed_certificates table: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE certificate_authorities 
MODIFY COLUMN type ENUM('digicert', 'ndes_scep_proxy', 'custom_scep_proxy', 'hydrant', 'smallstep') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
-- Smallstep fields
-- Note Smallstep also shares username and password fields with NDES
ADD COLUMN challenge_url TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL AFTER password_encrypted;`)
	if err != nil {
		return fmt.Errorf("failed to modify certificate_authorities table: %w", err)
	}
	return nil
}

func Down_20250923120000(tx *sql.Tx) error {
	return nil
}
