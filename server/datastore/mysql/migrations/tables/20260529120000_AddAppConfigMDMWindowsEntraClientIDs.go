package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260529120000, Down_20260529120000)
}

func Up_20260529120000(tx *sql.Tx) error {
	// Initialize windows_entra_client_ids to an empty array so GET /config returns [] (not null) on upgraded
	// installations and GitOps diffs stay stable, mirroring windows_entra_tenant_ids (migration 20260205184907).
	if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config != nil {
			config.MDM.WindowsEntraClientIDs = optjson.SetSlice([]string{})
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func Down_20260529120000(tx *sql.Tx) error {
	return nil
}
