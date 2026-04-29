package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260427134220, Down_20260427134220)
}

func Up_20260427134220(tx *sql.Tx) error {
	// Defaults to true for upgraded installations (where users already exist) so
	// that prior behavior is preserved, and false for fresh installations.
	var usersCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users;`).Scan(&usersCount); err != nil {
		return errors.Wrap(err, "select count users")
	}
	preserve := usersCount > 0

	if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config != nil {
			config.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment = preserve
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "set PreserveHostActivitiesOnReenrollment in AppConfig")
	}
	return nil
}

func Down_20260427134220(tx *sql.Tx) error {
	return nil
}
