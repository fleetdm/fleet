package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230320133602, Down_20230320133602)
}

func Up_20230320133602(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_disk_encryption_keys ADD COLUMN reset_requested TINYINT(1) NOT NULL DEFAULT 0`)
	return err
}

func Down_20230320133602(tx *sql.Tx) error {
	return nil
}
