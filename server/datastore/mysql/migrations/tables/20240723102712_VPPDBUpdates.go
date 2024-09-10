package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240723102712, Down_20240723102712)
}

func Up_20240723102712(tx *sql.Tx) error {
	_, err := tx.Exec(`
-- This table is the VPP equivalent of the "software_installers" table.
-- This table is also used as a cache of the response from the "Get Assets"
-- Apple endpoint as well as the FleetDM website endpoint which will return
-- the app metadata.
-- If an asset has an entry here and an entry in vpp_apps_teams, then it has
-- been added to Fleet.
CREATE TABLE vpp_apps (
	adam_id VARCHAR(16) NOT NULL,

	-- FK to the "software title" this app matches
	title_id int(10) unsigned DEFAULT NULL,

	bundle_identifier VARCHAR(255) NOT NULL DEFAULT '',
	icon_url VARCHAR(255) NOT NULL DEFAULT '',
	name VARCHAR(255) NOT NULL DEFAULT '',
	latest_version VARCHAR(255) NOT NULL DEFAULT '',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (adam_id),


	CONSTRAINT fk_vpp_apps_title
	  FOREIGN KEY (title_id)
	  REFERENCES software_titles (id)
	  ON DELETE SET NULL
	  ON UPDATE CASCADE
)`)
	if err != nil {
		return fmt.Errorf("failed to create table vpp_apps: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE vpp_apps_teams (
	id int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
	adam_id VARCHAR(16) NOT NULL,
	-- team_id NULL is for no team (cannot use 0 with foreign key)
	team_id INT(10) UNSIGNED NULL,

	-- this field is 0 for global, and the team_id otherwise, and is
	-- used for the unique index/constraint (team_id cannot be used
	-- as it allows NULL).
	global_or_team_id INT(10) NOT NULL DEFAULT 0,

	FOREIGN KEY (adam_id) REFERENCES vpp_apps (adam_id) ON DELETE CASCADE,
	FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
	UNIQUE KEY idx_global_or_team_id_adam_id (global_or_team_id, adam_id)
)`)
	if err != nil {
		return fmt.Errorf("failed to create table vpp_apps_teams: %w", err)
	}

	_, err = tx.Exec(`
-- This table is the VPP equivalent of the host_software_installs table.
-- It tracks the installation of VPP software on particular hosts.
CREATE TABLE host_vpp_software_installs (
	id int(10) unsigned NOT NULL AUTO_INCREMENT,
	host_id INT(10) UNSIGNED NOT NULL,

	-- This is the adam_id of the VPP software that's being installed
	adam_id VARCHAR(16) NOT NULL,

	-- This is the UUID of the MDM command issued to install the software
	command_uuid VARCHAR(127) NOT NULL,
	user_id INT(10) UNSIGNED NULL,

	-- This indicates whether or not this was a self-service install
	self_service TINYINT(1) NOT NULL DEFAULT FALSE,

	-- This is an ID for the event of "associating" the software with a host.
	-- This value comes from the "eventId" field in the response here:
	-- https://developer.apple.com/documentation/devicemanagement/associate_assets
	associated_event_id VARCHAR(36),

	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY(id),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL,
	FOREIGN KEY (adam_id) REFERENCES vpp_apps (adam_id) ON DELETE CASCADE,
	UNIQUE INDEX idx_host_vpp_software_installs_command_uuid (command_uuid)
)`)
	if err != nil {
		return fmt.Errorf("failed to create table host_vpp_software_installs: %w", err)
	}

	return nil
}

func Down_20240723102712(tx *sql.Tx) error {
	return nil
}
