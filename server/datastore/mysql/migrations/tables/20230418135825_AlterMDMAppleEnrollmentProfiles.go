package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230418135825, Down_20230418135825)
}

func Up_20230418135825(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE mdm_apple_enrollment_profiles DROP INDEX idx_type;
ALTER TABLE mdm_apple_enrollment_profiles ADD COLUMN team_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mdm_apple_enrollment_profiles ADD CONSTRAINT unq_enrollment_profiles_team_id_type UNIQUE (team_id, type);
`)
	return errors.Wrap(err, "add team_id")
}

func Down_20230418135825(tx *sql.Tx) error {
	return nil
}
