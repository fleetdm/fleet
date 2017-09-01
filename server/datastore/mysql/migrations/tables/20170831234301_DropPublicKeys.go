package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170831234301, Down_20170831234301)
}

func Up_20170831234301(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `public_keys`;")

	return err
}

func Down_20170831234301(tx *sql.Tx) error {
	return Down_20170131232841(tx)
}
