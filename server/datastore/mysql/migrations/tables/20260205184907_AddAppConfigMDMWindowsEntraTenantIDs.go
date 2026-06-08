package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260205184907, Down_20260205184907)
}

func Up_20260205184907(tx *sql.Tx) error {
	if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config != nil {
			config.MDM.WindowsEntraTenantIDs = optjson.SetSlice([]string{})
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func Down_20260205184907(tx *sql.Tx) error {
	return nil
}
