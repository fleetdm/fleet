package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240814142344, Down_20240814142344)
}

func Up_20240814142344(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE vpp_tokens (
	id                     int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	organization_name      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	location               varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	renew_at               timestamp NOT NULL,
	-- encrypted token, encrypted with the FleetConfig.Server.PrivateKey value
	token                  blob NOT NULL,

	team_id                int(10) UNSIGNED DEFAULT NULL,

	-- null_team_type indicates the special team represented when team_id is NULL,
	-- which can be none (VPP token is inactive), "all teams" or the "no team" team.
	-- It is important, when setting a non-NULL team_id, to always update this field
	-- to "none" so that if the team is deleted, via the ON DELETE SET NULL constraint, 
	-- the VPP token automatically becomes inactive (and not, e.g. "all teams" which 
	-- would introduce data inconsistency).
	null_team_type         ENUM('none', 'allteams', 'noteam') DEFAULT 'none',

	created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at             TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id),

	UNIQUE KEY idx_vpp_tokens_location (location),

	CONSTRAINT fk_vpp_tokens_team_id
		FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL
)`)
	if err != nil {
		return fmt.Errorf("failed to create table vpp_tokens: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_vpp_software_installs
	ADD COLUMN vpp_token_id int(10) UNSIGNED NULL,
	ADD CONSTRAINT fk_host_vpp_software_installs_vpp_token_id
		FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens(id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter table host_vpp_software_installs: %w", err)
	}
	return nil

	// TODO(mna): implement migration of existing VPP token...
}

func Down_20240814142344(tx *sql.Tx) error {
	return nil
}
