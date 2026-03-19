package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260124200020, Down_20260124200020)
}

func Up_20260124200020(tx *sql.Tx) error {
	_, err := tx.Exec(`
		UPDATE host_script_results
		SET attempt_number = 0
		WHERE attempt_number IS NULL AND exit_code IS NOT NULL
	`)
	if err != nil {
		return errors.Wrap(err, "cleanup null attempt_number in host_script_results")
	}

	_, err = tx.Exec(`
		UPDATE host_software_installs
		SET attempt_number = 0
		WHERE attempt_number IS NULL AND install_script_exit_code IS NOT NULL
	`)
	if err != nil {
		return errors.Wrap(err, "cleanup null attempt_number in host_software_installs")
	}

	return nil
}

func Down_20260124200020(tx *sql.Tx) error {
	return nil
}
