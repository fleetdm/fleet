package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260109231821, Down_20260109231821)
}

func Up_20260109231821(tx *sql.Tx) error {
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

	_, err = tx.Exec(`
		ALTER TABLE host_script_results
		ADD INDEX idx_host_script_results_host_policy_attempt (host_id, policy_id, attempt_number);
	`)
	if err != nil {
		return errors.Wrap(err, "adding index to host_script_results")
	}
	_, err = tx.Exec(`
		ALTER TABLE host_software_installs
		ADD INDEX idx_host_software_installs_host_policy_attempt (host_id, policy_id, attempt_number);
	`)
	if err != nil {
		return errors.Wrap(err, "adding index to host_software_installs")
	}

	return nil
}

func Down_20260109231821(tx *sql.Tx) error {
	return nil
}
