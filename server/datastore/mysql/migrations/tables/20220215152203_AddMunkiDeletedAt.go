package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220215152203, Down_20220215152203)
}

func Up_20220215152203(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE host_munki_info ADD COLUMN deleted_at timestamp NULL DEFAULT NULL;`); err != nil {
		return errors.Wrap(err, "adding host munki info deleted_at")
	}
	if _, err := tx.Exec(`UPDATE host_munki_info SET deleted_at=NOW() WHERE version=''`); err != nil {
		return errors.Wrap(err, "marking as deleted")
	}

	return nil
}

func Down_20220215152203(tx *sql.Tx) error {
	return nil
}
