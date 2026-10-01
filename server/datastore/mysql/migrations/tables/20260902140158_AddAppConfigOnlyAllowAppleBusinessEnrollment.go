package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260902140158, Down_20260902140158)
}

func Up_20260902140158(tx *sql.Tx) error {
	return updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		config.MDM.OnlyAllowAppleBusinessEnrollment = false
		return nil
	})
}

func Down_20260902140158(tx *sql.Tx) error {
	return nil
}
