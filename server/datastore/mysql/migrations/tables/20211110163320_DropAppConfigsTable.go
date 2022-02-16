package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211110163320, Down_20211110163320)
}

func Up_20211110163320(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE `app_configs`")
	if err != nil {
		return errors.Wrap(err, "drop app_configs table")
	}
	return nil
}

func Down_20211110163320(tx *sql.Tx) error {
	return nil
}
