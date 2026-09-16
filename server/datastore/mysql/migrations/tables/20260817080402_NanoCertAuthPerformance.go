package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260817080402, Down_20260817080402)
}

func Up_20260817080402(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE nano_cert_auth_associations ADD INDEX idx_sha256 (sha256)
	`)
	return err
}

func Down_20260817080402(tx *sql.Tx) error {
	return nil
}
