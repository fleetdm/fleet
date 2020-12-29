package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20201208121729, Down_20201208121729)
}

func Up_20201208121729(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `hosts` " +
			"ADD COLUMN `last_enroll_time` TIMESTAMP DEFAULT CURRENT_TIMESTAMP;",
	)
	if err != nil {
		return errors.Wrap(err, "add last_enroll_time column")
	}

	_, err = tx.Exec("UPDATE hosts SET last_enroll_time = created_at")
	if err != nil {
		return errors.Wrap(err, "set last_enroll_time")
	}

	return nil
}

func Down_20201208121729(tx *sql.Tx) error {
	return nil
}
