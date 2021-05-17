package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210513174845, Down_20210513174845)
}

func Up_20210513174845(tx *sql.Tx) error {
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

func Down_20210513174845(tx *sql.Tx) error {
	return nil
}
