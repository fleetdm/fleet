package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260318210346, Down_20260318210346)
}

func Up_20260318210346(tx *sql.Tx) error {
	return updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		// For existing instances, preserve current implicit behavior:
		// labels and secrets were already no-ops when omitted from GitOps.
		config.UIGitOpsMode.Exceptions.Labels = true
		config.UIGitOpsMode.Exceptions.Secrets = true
		config.UIGitOpsMode.Exceptions.Software = false
		return nil
	})
}

func Down_20260318210346(tx *sql.Tx) error {
	return nil
}
