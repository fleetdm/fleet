package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240515200020, Down_20240515200020)
}

func Up_20240515200020(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS software_installers (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,

  -- team_id NULL is for no team (cannot use 0 with foreign key)
  team_id INT(10) UNSIGNED NULL,
  -- this field is 0 for global, and the team_id otherwise, and is
  -- used for the unique index/constraint (team_id cannot be used
  -- as it allows NULL).
  global_or_team_id INT(10) UNSIGNED NOT NULL DEFAULT 0,

  -- FK to the "software title" this installer matches
  title_id int(10) unsigned DEFAULT NULL,

  -- Filename of the uploaded installer
  filename varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

  -- Version extracted from the uploaded installer
  version varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

  -- Platform extracted from the uploaded installer
  platform varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

  -- Raw osquery SQL statment to be run as a pre-install condition
  pre_install_query text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- FK to the script_contents for the script used to install this software
  install_script_content_id int(10) unsigned NOT NULL,

  -- FK to the script_contents for the post-script uploaded by the IT admin to
  -- be run after the software is installed
  post_install_script_content_id int(10) unsigned DEFAULT NULL,

  -- used to track the ID retrieved from the storage containing the installer bytes
  storage_id varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,

  uploaded_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (id),

  CONSTRAINT fk_software_installers_title
    FOREIGN KEY (title_id)
    REFERENCES software_titles (id)
    ON DELETE SET NULL
    ON UPDATE CASCADE,

  CONSTRAINT fk_software_installers_install_script_content_id
    FOREIGN KEY (install_script_content_id)
    REFERENCES script_contents (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,

  CONSTRAINT fk_software_installers_post_install_script_content_id
    FOREIGN KEY (post_install_script_content_id)
    REFERENCES script_contents (id)
    ON DELETE RESTRICT
    ON UPDATE CASCADE,

  CONSTRAINT fk_software_installers_team_id
    FOREIGN KEY (team_id)
    REFERENCES teams (id)
    ON DELETE CASCADE
    ON UPDATE CASCADE,

  UNIQUE KEY idx_software_installers_team_id_title_id (global_or_team_id, title_id),

  INDEX idx_software_installers_platform_title_id (platform, title_id)

)
  `)
	if err != nil {
		return fmt.Errorf("creating software_installers table: %w", err)
	}

	_, err = tx.Exec(`
-- this table tracks the status of a software installation in a host
CREATE TABLE IF NOT EXISTS host_software_installs (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,

  -- Unique identifier (e.g. UUID) generated for each
  -- install run.
  execution_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

  -- Soft reference to the hosts table, entries in this table are deleted in
  -- the application logic when a host is deleted.
  host_id int(10) unsigned NOT NULL,

  -- FK to the software installer that's being processed
  software_installer_id int(10) unsigned NOT NULL,

  -- Output of the osquery query used to determine if the installer should run.
  pre_install_query_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Output of the script used to install the software
  install_script_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Exit code of the script used to install the software
  install_script_exit_code int(10) DEFAULT NULL,

  -- Output of the post-script run after the software is installed
  post_install_script_output text COLLATE utf8mb4_unicode_ci DEFAULT NULL,

  -- Exit code of the post-script run after the software is installed
  post_install_script_exit_code int(10) DEFAULT NULL,

  -- User that requested the installation, for upcoming activities
  user_id int(10) unsigned DEFAULT NULL,

  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (id),

  CONSTRAINT fk_host_software_installs_installer_id
    FOREIGN KEY (software_installer_id)
    REFERENCES software_installers (id)
    ON DELETE CASCADE ON UPDATE CASCADE,

  CONSTRAINT fk_host_software_installs_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE SET NULL,

  KEY idx_host_software_installs_host_installer (host_id, software_installer_id),

  -- this index can be used to lookup results for a specific
  -- execution (execution ids, e.g. when updating the row for results)
  UNIQUE KEY idx_host_software_installs_execution_id (execution_id)
)
  `)
	if err != nil {
		return fmt.Errorf("creating host_software_installs table: %w", err)
	}

	return nil
}

func Down_20240515200020(tx *sql.Tx) error {
	return nil
}
