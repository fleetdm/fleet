package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210915144307, Down_20210915144307)
}

func Up_20210915144307(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE policies 
		ADD COLUMN team_id INT UNSIGNED,
		ADD FOREIGN KEY fk_policies_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	`); err != nil {
		return errors.Wrap(err, "add column team_id")
	}
	return nil
}

func Down_20210915144307(tx *sql.Tx) error {
	return nil
}
