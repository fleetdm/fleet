package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000001, Down_20210601000001)
}

func Up_20210601000001(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS teams (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		name VARCHAR(255) NOT NULL,
		description VARCHAR(1023) NOT NULL DEFAULT '',
		UNIQUE KEY idx_name (name)
	)`); err != nil {
		return errors.Wrap(err, "create teams")
	}

	// Users <> Teams mapping
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS user_teams (
		user_id INT UNSIGNED NOT NULL,
		team_id INT UNSIGNED NOT NULL,
		role VARCHAR(64) NOT NULL,
		PRIMARY KEY (user_id, team_id),
		FOREIGN KEY fk_user_teams_user_id (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
		FOREIGN KEY fk_user_teams_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	)`); err != nil {
		return errors.Wrap(err, "create user_teams")
	}

	if _, err := tx.Exec(`ALTER TABLE hosts
		ADD team_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_hosts_team_id (team_id) REFERENCES teams (id) ON DELETE SET NULL
	`); err != nil {
		return errors.Wrap(err, "alter hosts")
	}

	if _, err := tx.Exec(`ALTER TABLE users
		ADD global_role VARCHAR(64) DEFAULT NULL
	`); err != nil {
		return errors.Wrap(err, "alter users")
	}

	return nil
}

func Down_20210601000001(tx *sql.Tx) error {
	return nil
}
