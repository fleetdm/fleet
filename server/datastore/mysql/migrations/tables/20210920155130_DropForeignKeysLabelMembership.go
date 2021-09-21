package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210920155130, Down_20210920155130)
}

func Up_20210920155130(tx *sql.Tx) error {
	_, err := tx.Exec(
		`ALTER TABLE label_membership DROP FOREIGN KEY fk_lm_host_id, DROP FOREIGN KEY fk_lm_label_id;`)
	if err != nil {
		return errors.Wrap(err, "dropping foreign keys for label_membership")
	}
	return nil
}

func Down_20210920155130(tx *sql.Tx) error {
	return nil
}
