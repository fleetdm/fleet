package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260316120004, Down_20260316120004)
}

func Up_20260316120004(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStepWithArgs(
			"UPDATE host_mdm_windows_profiles SET status = ? WHERE status = ?",
			[]any{fleet.MDMDeliveryVerified, fleet.MDMDeliveryVerifying},
			"failed to update host_mdm_windows_profiles from verifying",
		),
		basicMigrationStepWithArgs(
			"UPDATE host_mdm_windows_profiles SET status = ?, detail = '' WHERE status = ? AND detail = ?",
			[]any{fleet.MDMDeliveryVerified, fleet.MDMDeliveryFailed, fleet.HostMDMProfileDetailFailedWasVerifying},
			"failed to update host_mdm_windows_profiles from failed with non-verifying detail",
		),
	}, tx)
}

func Down_20260316120004(tx *sql.Tx) error {
	return nil
}
