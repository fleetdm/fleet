package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170110202752, Down_20170110202752)
}

func Up_20170110202752(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"DROP COLUMN `smtp_enabled`;",
	)
	return err
}

func Down_20170110202752(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `smtp_enabled` TINYINT(1) NOT NULL DEFAULT FALSE;",
	)
	return err
}
