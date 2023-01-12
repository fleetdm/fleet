package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221220195934, Down_20221220195934)
}

func Up_20221220195934(tx *sql.Tx) error {
	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM scep_serials").Scan(&count)
	if err != nil {
		return errors.Wrap(err, "count scep_serials")
	}

	// if the database already has serials, don't change the auto
	// increment.
	if count > 0 {
		return nil
	}

	// Start assigning serials from 2, as we assume the first serial
	// is issued to the CA.
	//
	// See https://github.com/fleetdm/fleet/issues/8167 for more
	// details.
	_, err = tx.Exec("ALTER TABLE `scep_serials` AUTO_INCREMENT = 2")
	return errors.Wrap(err, "set scep_serials auto increment")
}

func Down_20221220195934(tx *sql.Tx) error {
	return nil
}
