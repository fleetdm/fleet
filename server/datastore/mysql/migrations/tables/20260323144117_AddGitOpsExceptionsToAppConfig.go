package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260323144117, Down_20260323144117)
}

func Up_20260323144117(tx *sql.Tx) error {
	return updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		// For existing instances, preserve current implicit behavior:
		// labels and secrets were already no-ops when omitted from GitOps.
		config.GitOpsConfig.Exceptions.Labels = true
		config.GitOpsConfig.Exceptions.Secrets = true
		config.GitOpsConfig.Exceptions.Software = false
		return nil
	})
}

func Down_20260323144117(tx *sql.Tx) error {
	return nil
}
