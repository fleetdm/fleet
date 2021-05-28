package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20210527151852, Down_20210527151852)
}

func Up_20210527151852(tx *sql.Tx) error {
	sql := `
		ALTER TABLE enroll_secrets
		ADD COLUMN team_id INT UNSIGNED,
		ADD FOREIGN KEY fk_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add team_id to enroll_secrets")
	}

	return nil
}

func Down_20210527151852(tx *sql.Tx) error {
	return nil
}
