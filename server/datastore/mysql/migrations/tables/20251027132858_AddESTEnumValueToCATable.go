package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251027132858, Down_20251027132858)
}

func Up_20251027132858(tx *sql.Tx) error {
	tx.Exec(`
ALTER TABLE certificate_authorities
MODIFY COLUMN type
ENUM('digicert','ndes_scep_proxy','custom_scep_proxy','hydrant','smallstep','custom_est_proxy') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL
`)
	return nil
}

func Down_20251027132858(tx *sql.Tx) error {
	return nil
}
