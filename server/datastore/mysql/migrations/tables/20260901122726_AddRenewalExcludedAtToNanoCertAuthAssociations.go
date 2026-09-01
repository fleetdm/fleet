package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260901122726, Down_20260901122726)
}

func Up_20260901122726(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE nano_cert_auth_associations ADD COLUMN renewal_excluded_at TIMESTAMP(6) NULL DEFAULT NULL;")
	return err
}

func Down_20260901122726(tx *sql.Tx) error {
	return nil
}
