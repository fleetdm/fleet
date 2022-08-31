package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220608113128, Down_20220608113128)
}

func Up_20220608113128(tx *sql.Tx) error {
	err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config.FleetDesktop.TransparencyURL != "" {
			return errors.New("unexpected transparency_url value in app_config_json")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func Down_20220608113128(tx *sql.Tx) error {
	return nil
}
