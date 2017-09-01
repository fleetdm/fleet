package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170831234300, Down_20170831234300)
}

func Up_20170831234300(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `licenses`;")

	return err
}

func Down_20170831234300(tx *sql.Tx) error {
	return Up_20170127014618(tx)
}
