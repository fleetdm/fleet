package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260717152653, Down_20260717152653)
}

// Up_20260717152653 corrects the precedence of the generated status and
// execution_status columns on host_software_installs. Previously a post-install
// script that exited 0 reported the install as "installed" even when the install
// script itself exited non-zero. Because fleetd runs the post-install script
// regardless of the install script's outcome, that masked failed installs as
// successful. A non-zero install-script exit code is now terminal (failed_install)
// and evaluated before the post-install script result.
func Up_20260717152653(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_software_installs
		MODIFY COLUMN ` + "`status`" + ` ENUM('pending_install','failed_install','installed','pending_uninstall','failed_uninstall','canceled_install','canceled_uninstall')
		GENERATED ALWAYS AS (
			CASE
				WHEN removed = 1 THEN NULL
				WHEN canceled = 1 AND uninstall = 0 THEN 'canceled_install'
				WHEN canceled = 1 AND uninstall = 1 THEN 'canceled_uninstall'
				WHEN install_script_exit_code IS NOT NULL AND install_script_exit_code != 0 THEN 'failed_install'
				WHEN post_install_script_exit_code IS NOT NULL AND post_install_script_exit_code = 0 THEN 'installed'
				WHEN post_install_script_exit_code IS NOT NULL AND post_install_script_exit_code != 0 THEN 'failed_install'
				WHEN install_script_exit_code IS NOT NULL AND install_script_exit_code = 0 THEN 'installed'
				WHEN pre_install_query_output IS NOT NULL AND pre_install_query_output = '' THEN 'failed_install'
				WHEN host_id IS NOT NULL AND uninstall = 0 THEN 'pending_install'
				WHEN uninstall_script_exit_code IS NOT NULL AND uninstall_script_exit_code != 0 THEN 'failed_uninstall'
				WHEN uninstall_script_exit_code IS NOT NULL AND uninstall_script_exit_code = 0 THEN NULL
				WHEN host_id IS NOT NULL AND uninstall = 1 THEN 'pending_uninstall'
				ELSE NULL
			END
		) STORED
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_software_installs
		MODIFY COLUMN ` + "`execution_status`" + ` ENUM('pending_install','failed_install','installed','pending_uninstall','failed_uninstall','canceled_install','canceled_uninstall')
		GENERATED ALWAYS AS (
			CASE
				WHEN canceled = 1 AND uninstall = 0 THEN 'canceled_install'
				WHEN canceled = 1 AND uninstall = 1 THEN 'canceled_uninstall'
				WHEN install_script_exit_code IS NOT NULL AND install_script_exit_code != 0 THEN 'failed_install'
				WHEN post_install_script_exit_code IS NOT NULL AND post_install_script_exit_code = 0 THEN 'installed'
				WHEN post_install_script_exit_code IS NOT NULL AND post_install_script_exit_code != 0 THEN 'failed_install'
				WHEN install_script_exit_code IS NOT NULL AND install_script_exit_code = 0 THEN 'installed'
				WHEN pre_install_query_output IS NOT NULL AND pre_install_query_output = '' THEN 'failed_install'
				WHEN host_id IS NOT NULL AND uninstall = 0 THEN 'pending_install'
				WHEN uninstall_script_exit_code IS NOT NULL AND uninstall_script_exit_code != 0 THEN 'failed_uninstall'
				WHEN uninstall_script_exit_code IS NOT NULL AND uninstall_script_exit_code = 0 THEN NULL
				WHEN host_id IS NOT NULL AND uninstall = 1 THEN 'pending_uninstall'
				ELSE NULL
			END
		) VIRTUAL
	`); err != nil {
		return err
	}

	return nil
}

func Down_20260717152653(tx *sql.Tx) error {
	return nil
}
