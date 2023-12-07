package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20231009094542, Down_20231009094542)
}

func Up_20231009094542(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE scripts ADD UNIQUE KEY idx_scripts_team_name (team_id, name)`)
	return err
}

func Down_20231009094542(tx *sql.Tx) error {
	return nil
}
