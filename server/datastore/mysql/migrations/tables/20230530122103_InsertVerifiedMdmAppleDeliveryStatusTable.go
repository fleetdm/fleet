package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230530122103, Down_20230530122103)
}

func Up_20230530122103(tx *sql.Tx) error {
	_, err := tx.Exec(`INSERT INTO mdm_apple_delivery_status (status) VALUES(?)`, "verified")
	if err != nil {
		return errors.Wrap(err, "insert verified mdm_apple_delivery_status")
	}

	return nil
}

func Down_20230530122103(tx *sql.Tx) error {
	return nil
}
