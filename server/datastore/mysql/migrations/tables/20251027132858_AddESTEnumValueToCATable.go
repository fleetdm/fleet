package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251027132858, Down_20251027132858)
}

func Up_20251027132858(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE certificate_authorities
		MODIFY COLUMN type
		ENUM('digicert','ndes_scep_proxy','custom_scep_proxy','hydrant','smallstep','custom_est_proxy') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL
    `); err != nil {
		return fmt.Errorf("adding custom_est_proxy to certificate_authorities.types enum: %w", err)
	}
	return nil
}

func Down_20251027132858(tx *sql.Tx) error {
	return nil
}
