package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230317173844, Down_20230317173844)
}

func Up_20230317173844(tx *sql.Tx) error {
	_, err := tx.Exec(`
DELETE FROM host_mdm_apple_profiles
WHERE operation_type = 'remove'
	AND status = 'failed'
	AND INSTR(detail, 'MDMClientError (89)') > 0`)
	if err != nil {
		return errors.Wrap(err, "cleanup host mdm apple profiles")
	}

	return nil
}

func Down_20230317173844(tx *sql.Tx) error {
	return nil
}
