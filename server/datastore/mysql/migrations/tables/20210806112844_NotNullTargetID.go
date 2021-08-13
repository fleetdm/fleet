package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210806112844, Down_20210806112844)
}

func Up_20210806112844(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM pack_targets WHERE target_id is NULL`); err != nil {
		return errors.Wrap(err, "delete target_id null pack targets")
	}

	if _, err := tx.Exec(`ALTER TABLE pack_targets MODIFY target_id int unsigned NOT NULL`); err != nil {
		return errors.Wrap(err, "make pack_targets.target_id not null")
	}
	return nil
}

func Down_20210806112844(tx *sql.Tx) error {
	return nil
}
