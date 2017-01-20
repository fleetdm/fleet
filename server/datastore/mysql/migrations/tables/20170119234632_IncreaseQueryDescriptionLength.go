package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170119234632, Down_20170119234632)
}

func Up_20170119234632(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `queries` MODIFY `description` TEXT NOT NULL;",
	)
	return err
}

func Down_20170119234632(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `queries` MODIFY `description` VARCHAR(255) NOT NULL;",
	)
	return err
}
