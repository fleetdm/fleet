package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240829170023, Down_20240829170023)
}

func Up_20240829170023(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE vpp_token_teams (
	id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
    vpp_token_id int unsigned NOT NULL,
	team_id int unsigned,
	null_team_type enum('none','allteams','noteam') COLLATE utf8mb4_unicode_ci DEFAULT 'none',
	UNIQUE KEY idx_vpp_token_teams_team_id (team_id),
	-- Note that this is only a partial constraint. There can be only
	-- one token per team, but the team "No team" and "all teams" have
	-- to be checked manually in go code
	CONSTRAINT fk_vpp_token_teams_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
	CONSTRAINT fk_vpp_token_teams_vpp_token_id FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens (id) ON DELETE CASCADE
);

INSERT INTO vpp_token_teams (
	vpp_token_id,
	team_id,
	null_team_type
) SELECT
	id,
	team_id,
	null_team_type
FROM vpp_tokens;

ALTER TABLE vpp_tokens DROP FOREIGN KEY fk_vpp_tokens_team_id;
ALTER TABLE vpp_tokens DROP CONSTRAINT idx_vpp_tokens_team_id;
ALTER TABLE vpp_tokens DROP COLUMN team_id;
ALTER TABLE vpp_tokens DROP COLUMN null_team_type;
`)
	if err != nil {
		return fmt.Errorf("migrating vpp_tokens associations to join table: %w", err)
	}

	return nil
}

func Down_20240829170023(tx *sql.Tx) error {
	return nil
}
