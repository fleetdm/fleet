package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261020200751, Down_20261020200751)
}

func Up_20261020200751(tx *sql.Tx) error {
	var err error
	if !columnExists(tx, "vpp_apps_teams", "configuration_updated_at") {
		_, err = tx.Exec(`ALTER TABLE vpp_apps_teams ADD COLUMN configuration_updated_at DATETIME(6) NULL DEFAULT NULL`)
		if err != nil {
			return fmt.Errorf("adding configuration_updated_at to vpp_apps_teams: %w", err)
		}
	}

	if !columnExists(tx, "host_vpp_software_installs", "needs_configuration_resend") {
		_, err = tx.Exec(`ALTER TABLE host_vpp_software_installs ADD COLUMN needs_configuration_resend TINYINT(1) NOT NULL DEFAULT '0'`)
		if err != nil {
			return fmt.Errorf("adding needs_configuration_resend to host_vpp_software_installs: %w", err)
		}
	}

	// Link existing iOS and iPadOS configurations to their Fleet variables, matching up to the end of the name so USERNAME doesn't match USERNAME_LOCAL_PART
	_, err = tx.Exec(`
		INSERT IGNORE INTO mdm_configuration_profile_variables (vpp_app_team_id, fleet_variable_id)
		SELECT vat.id, fv.id
		FROM vpp_apps_teams vat
		JOIN fleet_variables fv
			ON fv.is_prefix = 0 AND vat.configuration REGEXP CONCAT('\\$\\{?', fv.name, '([^A-Za-z0-9_]|$)')
		WHERE vat.platform IN ('ios', 'ipados') AND vat.configuration IS NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("linking app store app configurations to fleet variables: %w", err)
	}

	return nil
}

func Down_20261020200751(tx *sql.Tx) error {
	return nil
}
