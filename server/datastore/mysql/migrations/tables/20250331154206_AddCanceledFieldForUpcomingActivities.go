package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250331154206, Down_20250331154206)
}

func Up_20250331154206(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE host_script_results
	ADD COLUMN canceled TINYINT(1) NOT NULL DEFAULT '0'
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_script_results: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_vpp_software_installs
	ADD COLUMN canceled TINYINT(1) NOT NULL DEFAULT '0'
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_vpp_software_installs: %w", err)
	}

	if _, err := tx.Exec(`
ALTER TABLE host_software_installs
	ADD COLUMN canceled TINYINT(1) NOT NULL DEFAULT '0',
	CHANGE COLUMN execution_status execution_status 
		ENUM('pending_install', 'failed_install', 'installed', 'pending_uninstall', 'failed_uninstall', 'canceled_install', 'canceled_uninstall')
GENERATED ALWAYS AS (
CASE
	WHEN canceled = 1 AND uninstall = 0 THEN 'canceled_install'

	WHEN canceled = 1 AND uninstall = 1 THEN 'canceled_uninstall'

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
) VIRTUAL NULL,

	CHANGE COLUMN status status 
		ENUM('pending_install', 'failed_install', 'installed', 'pending_uninstall', 'failed_uninstall', 'canceled_install', 'canceled_uninstall')
GENERATED ALWAYS AS (
CASE
	WHEN removed = 1 THEN NULL

	WHEN canceled = 1 AND uninstall = 0 THEN 'canceled_install'

	WHEN canceled = 1 AND uninstall = 1 THEN 'canceled_uninstall'

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
) STORED NULL
`); err != nil {
		return fmt.Errorf("failed to add canceled column and update statuses generated columns on host_software_installs: %w", err)
	}
	return nil
}

func Down_20250331154206(tx *sql.Tx) error {
	return nil
}
