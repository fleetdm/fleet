package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240424124712, Down_20240424124712)
}

func Up_20240424124712(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS software_installers (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,

  -- FK to the "software version" this installer matches
  software_id bigint(20) unsigned DEFAULT NULL,

  -- Raw osquery SQL statment to be run as a pre-install condition
  pre_install_condition text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- FK to the script_contents for the script used to install this software
  install_script_id int(10) unsigned NOT NULL,

  -- FK to the script_contents for the post-script uploaded by the IT admin to
  -- be run after the software is installed 
  post_install_script_id int(10) unsigned DEFAULT NULL,

  PRIMARY KEY (id),

  CONSTRAINT fk_software_installers_version
    FOREIGN KEY (software_id)
    REFERENCES software (id)
    ON DELETE SET NULL
    ON UPDATE CASCADE,

  CONSTRAINT fk_software_installers_install_script_id
    FOREIGN KEY (install_script_id)
    REFERENCES script_contents (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,

  CONSTRAINT fk_software_installers_post_install_script_id
    FOREIGN KEY (post_install_script_id)
    REFERENCES script_contents (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE
)
  `)
	if err != nil {
		return fmt.Errorf("creating software_installers table: %w", err)
	}

	_, err = tx.Exec(`
-- this table tracks the status of a software installation in a host
CREATE TABLE IF NOT EXISTS host_software_installs (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,

  -- Soft reference to the hosts table, entries in this table are deleted in
  -- the application logic when a host is deleted.
  host_id int(10) unsigned NOT NULL,

  -- FK to the software installer that's being processed
  software_installer_id int(10) unsigned NOT NULL,

  -- Output of the osquery query used to determine if the installer should run.
  pre_install_condition_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Output of the script used to install the software
  install_script_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Exit code of the script used to install the software
  install_script_exit_code int(10) DEFAULT NULL,

  -- Output of the post-script run after the software is installed
  post_install_condition_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Exit code of the post-script run after the software is installed
  post_install_condition_exit_code int(10) DEFAULT NULL,

  PRIMARY KEY (id),

  CONSTRAINT fk_host_software_installs_installer_id
    FOREIGN KEY (software_installer_id)
    REFERENCES software_installers (id)
    ON DELETE CASCADE ON UPDATE CASCADE,

  UNIQUE KEY idx_host_software_installs_host_installer (host_id, software_installer_id)
)
  `)
	if err != nil {
		return fmt.Errorf("creating host_software_installs table: %w", err)
	}

	return nil
}

func Down_20240424124712(tx *sql.Tx) error {
	return nil
}
