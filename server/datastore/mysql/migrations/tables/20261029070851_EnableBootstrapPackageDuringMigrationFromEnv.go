package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20261029070851, Down_20261029070851)
}

// Up_20261029070851 carries over the deprecated FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION env var
// into the macos_enable_bootstrap_package_during_migration setting of no team and the teams that have a
// bootstrap package, since the setting can't be enabled without one.
func Up_20261029070851(tx *sql.Tx) error {
	if !fleet.BootstrapPackageDuringMigrationEnvEnabled() {
		return nil
	}

	var noTeamHasPackage bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM mdm_apple_bootstrap_packages WHERE team_id = 0)`).Scan(&noTeamHasPackage); err != nil {
		return fmt.Errorf("checking no team bootstrap package: %w", err)
	}
	if noTeamHasPackage {
		if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
			config.MDM.MacOSSetup.MacOSEnableBootstrapPackageDuringMigration = true
			return nil
		}); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
UPDATE teams
SET config = JSON_SET(config, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration', CAST('true' AS JSON))
WHERE config IS NOT NULL AND id IN (SELECT team_id FROM mdm_apple_bootstrap_packages)`); err != nil {
		return fmt.Errorf("enabling bootstrap package during migration for teams: %w", err)
	}

	return nil
}

func Down_20261029070851(tx *sql.Tx) error {
	return nil
}
