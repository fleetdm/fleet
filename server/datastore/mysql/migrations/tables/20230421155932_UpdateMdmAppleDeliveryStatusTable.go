package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230421155932, Down_20230421155932)
}

func Up_20230421155932(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE mdm_apple_delivery_status SET status = 'verifying' WHERE status = 'applied'`)
	if err != nil {
		return errors.Wrap(err, "update mdm_apple_delivery_status")
	}

	return nil
}

func Down_20230421155932(tx *sql.Tx) error {
	return nil
}
