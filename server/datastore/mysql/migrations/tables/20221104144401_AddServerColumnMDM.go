package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20221104144401, Down_20221104144401)
}

func Up_20221104144401(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_mdm ADD COLUMN is_server TINYINT(1) NULL;`)
	return err
}

func Down_20221104144401(tx *sql.Tx) error {
	return nil
}
