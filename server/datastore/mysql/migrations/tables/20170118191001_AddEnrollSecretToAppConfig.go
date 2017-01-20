package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170118191001, Down_20170118191001)
}

func Up_20170118191001(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `osquery_enroll_secret` VARCHAR(255) NOT NULL DEFAULT '';",
	)
	return err
}

func Down_20170118191001(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"DROP COLUMN `osquery_enroll_secret`;",
	)
	return err
}
