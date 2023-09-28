package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230726115701, Down_20230726115701)
}

func Up_20230726115701(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE queries
			ADD UNIQUE INDEX idx_name_team_id_unq (name, team_id_char),
			ADD INDEX idx_team_id_saved_auto_interval (team_id, saved, automations_enabled, schedule_interval);
	`); err != nil {
		return errors.Wrap(err, "updating queries indices")
	}
	return nil
}

func Down_20230726115701(tx *sql.Tx) error {
	return nil
}
