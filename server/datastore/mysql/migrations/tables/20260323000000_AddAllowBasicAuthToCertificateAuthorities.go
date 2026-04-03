package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260323000000, Down_20260323000000)
}

func Up_20260323000000(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE certificate_authorities ADD COLUMN allow_basic_auth TINYINT(1) NOT NULL DEFAULT 0 AFTER admin_url`)
	return err
}

func Down_20260323000000(tx *sql.Tx) error {
	return nil
}
