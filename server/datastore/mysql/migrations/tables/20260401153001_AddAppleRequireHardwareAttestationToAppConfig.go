package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260401153001, Down_20260401153001)
}

func Up_20260401153001(tx *sql.Tx) error {
	err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config != nil {
			config.MDM.AppleRequireHardwareAttestation = false
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "set AppleRequireHardwareAttestation in AppConfig")
	}
	return nil
}

func Down_20260401153001(tx *sql.Tx) error {
	return nil
}
