package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241021224359, Down_20241021224359)
}

func Up_20241021224359(tx *sql.Tx) error {
	// Column is added for "status irrespective of whether removed flag is set", for showing details on a single
	// (un)install. The normal status column is used for aggregate metrics of "how many hosts have this installed",
	// so it needs to be reset when installers change.
	if _, err := tx.Exec(`
ALTER TABLE host_software_installs
ADD COLUMN execution_status ENUM('pending_install', 'failed_install', 'installed', 'pending_uninstall', 'failed_uninstall')
GENERATED ALWAYS AS (
CASE
	WHEN post_install_script_exit_code IS NOT NULL AND
		post_install_script_exit_code = 0 THEN 'installed'

	WHEN post_install_script_exit_code IS NOT NULL AND
		post_install_script_exit_code != 0 THEN 'failed_install'

	WHEN install_script_exit_code IS NOT NULL AND
		install_script_exit_code = 0 THEN 'installed'

	WHEN install_script_exit_code IS NOT NULL AND
		install_script_exit_code != 0 THEN 'failed_install'

	WHEN pre_install_query_output IS NOT NULL AND
		pre_install_query_output = '' THEN 'failed_install'

	WHEN host_id IS NOT NULL AND uninstall = 0 THEN 'pending_install'

	WHEN uninstall_script_exit_code IS NOT NULL AND
		uninstall_script_exit_code != 0 THEN 'failed_uninstall'

	WHEN uninstall_script_exit_code IS NOT NULL AND
		uninstall_script_exit_code = 0 THEN NULL -- available for install again

	WHEN host_id IS NOT NULL AND uninstall = 1 THEN 'pending_uninstall'

	ELSE NULL -- not installed from Fleet installer or successfully uninstalled
END
) VIRTUAL NULL`); err != nil {
		return fmt.Errorf("failed to add execution_status column to host_software_installs: %w", err)
	}

	return nil
}

func Down_20241021224359(tx *sql.Tx) error {
	return nil
}
