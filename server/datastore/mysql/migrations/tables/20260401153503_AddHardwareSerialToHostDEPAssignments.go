package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260401153503, Down_20260401153503)
}

func Up_20260401153503(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE host_dep_assignments
			ADD COLUMN hardware_serial varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
	`)
	if err != nil {
		return errors.Wrap(err, "add host_dep_assignments.hardware_serial column")
	}

	return nil
}

func Down_20260401153503(tx *sql.Tx) error {
	return nil
}
