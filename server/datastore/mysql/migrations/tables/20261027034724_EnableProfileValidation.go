package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20261027034724, Down_20261027034724)
}

func Up_20261027034724(tx *sql.Tx) error {
	return updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		config.MDM.EnableProfileValidation = true
		return nil
	})
}

func Down_20261027034724(tx *sql.Tx) error {
	return nil
}
