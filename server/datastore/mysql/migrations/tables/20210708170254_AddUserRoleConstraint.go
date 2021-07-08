package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20210708170254, Down_20210708170254)
}

func Up_20210708170254(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD CHECK (global_role IN ('admin', 'maintainer', 'observer'));`)
	return err
}

func Down_20210708170254(tx *sql.Tx) error {
	return nil
}
