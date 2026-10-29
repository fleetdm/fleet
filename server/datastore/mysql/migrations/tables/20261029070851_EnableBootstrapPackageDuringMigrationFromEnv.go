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
// into the macos_enable_bootstrap_package_during_migration setting of no team and every team. The env
// var check is inlined (instead of using the fleet package helper) so this keeps working once the env
// var support is removed.
func Up_20261029070851(tx *sql.Tx) error {
	if !fleet.BootstrapPackageDuringMigrationEnvEnabled() {
		return nil
	}

	if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		config.MDM.MacOSSetup.MacOSEnableBootstrapPackageDuringMigration = true
		return nil
	}); err != nil {
		return err
	}

	if _, err := tx.Exec(`
UPDATE teams
SET config = JSON_SET(config, '$.mdm.macos_setup.macos_enable_bootstrap_package_during_migration', true)
WHERE config IS NOT NULL`); err != nil {
		return fmt.Errorf("enabling bootstrap package during migration for teams: %w", err)
	}

	return nil
}

func Down_20261029070851(tx *sql.Tx) error {
	return nil
}
