package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170504130602, Down_20170504130602)
}

func Up_20170504130602(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE `invites` ADD COLUMN `sso_enabled` TINYINT(1) NOT NULL DEFAULT FALSE AFTER `token`;")
	return err
}

func Down_20170504130602(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE `invites` DROP COLUMN `sso_enabled`;")
	return err
}
