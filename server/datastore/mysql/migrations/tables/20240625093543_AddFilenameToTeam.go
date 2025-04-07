package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240625093543, Down_20240625093543)
}

func Up_20240625093543(tx *sql.Tx) error {
	_, err := tx.Exec(
		`
		ALTER TABLE teams
		ADD COLUMN filename VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
		ADD CONSTRAINT idx_teams_filename UNIQUE INDEX (filename)`,
	)
	if err != nil {
		return fmt.Errorf("failed to add filename column to teams: %w", err)
	}
	return nil
}

func Down_20240625093543(_ *sql.Tx) error {
	return nil
}
