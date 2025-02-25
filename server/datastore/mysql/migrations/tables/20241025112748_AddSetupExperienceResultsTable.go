package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241025112748, Down_20241025112748)
}

func Up_20241025112748(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE setup_experience_scripts (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	team_id INT UNSIGNED DEFAULT NULL,
	global_or_team_id INT UNSIGNED NOT NULL DEFAULT '0',
	name VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	script_content_id INT UNSIGNED DEFAULT NULL,

	PRIMARY KEY (id),

	UNIQUE KEY idx_setup_experience_scripts_global_or_team_id (global_or_team_id),

	KEY idx_script_content_id (script_content_id),

	CONSTRAINT fk_setup_experience_scripts_ibfk_1 FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT fk_setup_experience_scripts_ibfk_2 FOREIGN KEY (script_content_id) REFERENCES script_contents (id) ON DELETE CASCADE
);

`)
	if err != nil {
		return fmt.Errorf("failed to create setup_experience_scripts table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_script_results ADD setup_experience_script_id INT UNSIGNED DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add setup_experience_scripts_id key to host_script_results: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_script_results
	ADD CONSTRAINT fk_host_script_results_setup_experience_id
	FOREIGN KEY (setup_experience_script_id)
	REFERENCES setup_experience_scripts (id) ON DELETE SET NULL`)
	if err != nil {
		return fmt.Errorf("failed to add foreign key constraint for host_script_resutls setup_experience column: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE setup_experience_status_results (
	id		INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	host_uuid	VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	name		VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	status		ENUM('pending', 'running', 'success', 'failure') NOT NULL,

	-- Software installer reference
	software_installer_id INT(10) UNSIGNED,
	-- Software installs reference
	host_software_installs_execution_id VARCHAR(255),

	-- VPP app reference
	vpp_app_team_id INT(10) UNSIGNED,
	-- VPP app install reference
	nano_command_uuid VARCHAR(255) COLLATE utf8mb4_unicode_ci,

	-- Setup script reference
	setup_experience_script_id INT(10) UNSIGNED,
	-- Script execution reference
	script_execution_id	VARCHAR(255) COLLATE utf8mb4_unicode_ci,
	error 		VARCHAR(255) COLLATE utf8mb4_unicode_ci,


	PRIMARY KEY (id),

	KEY idx_setup_experience_scripts_host_uuid (host_uuid),
	KEY idx_setup_experience_scripts_hsi_id (host_software_installs_execution_id),
	KEY idx_setup_experience_scripts_nano_command_uuid (nano_command_uuid),
	KEY idx_setup_experience_scripts_script_execution_id (script_execution_id),

	CONSTRAINT fk_setup_experience_status_results_si_id FOREIGN KEY (software_installer_id) REFERENCES software_installers(id) ON DELETE CASCADE,
	CONSTRAINT fk_setup_experience_status_results_va_id FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams(id) ON DELETE CASCADE,
	CONSTRAINT fk_setup_experience_status_results_ses_id FOREIGN KEY (setup_experience_script_id) REFERENCES setup_experience_scripts(id) ON DELETE CASCADE
)
`)
	// Service layer state machine like SetupExperienceNestStep()?
	// Called from each of the three endpoints (software install, vpp
	// mdm, scripts) involved in the setup when an eligible installer
	// writes its results
	if err != nil {
		return fmt.Errorf("failed to create setup_experience_status_results table: %w", err)
	}

	return nil
}

func Down_20241025112748(tx *sql.Tx) error {
	return nil
}
