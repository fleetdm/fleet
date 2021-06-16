package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000002, Down_20210601000002)
}

func Up_20210601000002(tx *sql.Tx) error {
	// Invites <> Teams mapping
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS invite_teams (
		invite_id INT UNSIGNED NOT NULL,
		team_id INT UNSIGNED NOT NULL,
		role VARCHAR(64) NOT NULL,
		PRIMARY KEY (invite_id, team_id),
		FOREIGN KEY fk_invite_id (invite_id) REFERENCES invites (id) ON DELETE CASCADE ON UPDATE CASCADE,
		FOREIGN KEY fk_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	)`); err != nil {
		return errors.Wrap(err, "create invite_teams")
	}

	if _, err := tx.Exec(`ALTER TABLE invites
		ADD global_role VARCHAR(64) DEFAULT NULL
	`); err != nil {
		return errors.Wrap(err, "alter users")
	}

	return nil
}

func Down_20210601000002(tx *sql.Tx) error {
	return nil
}
