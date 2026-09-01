package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260901122716, Down_20260901122716)
}

func Up_20260901122716(tx *sql.Tx) error {
	return updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		config.MDM.OnlyAllowAppleBusinessEnrollment = false
		return nil
	})
}

func Down_20260901122716(tx *sql.Tx) error {
	return nil
}
