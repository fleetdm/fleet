package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260106164935, Down_20260106164935)
}

func Up_20260106164935(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE host_script_results
		ADD COLUMN attempt_number INT DEFAULT NULL;
	`)
	if err != nil {
		return errors.Wrap(err, "adding attempt_number column to host_script_results")
	}
	_, err = tx.Exec(`
		ALTER TABLE host_software_installs
		ADD COLUMN attempt_number INT DEFAULT NULL;
	`)
	if err != nil {
		return errors.Wrap(err, "adding attempt_number column to host_software_installs")
	}

	return nil
}

func Down_20260106164935(tx *sql.Tx) error {
	return nil
}
