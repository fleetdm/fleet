package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20200504120000, Down_20200504120000)
}

func Up_20200504120000(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `hosts` " +
			"ADD COLUMN `additional` JSON DEFAULT NULL;",
	)
	if err != nil {
		return errors.Wrap(err, "add additional column")
	}

	_, err = tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `additional_queries` JSON DEFAULT NULL;",
	)
	if err != nil {
		return errors.Wrap(err, "add additional_queries column")
	}

	return nil
}

func Down_20200504120000(tx *sql.Tx) error {
	return nil
}
