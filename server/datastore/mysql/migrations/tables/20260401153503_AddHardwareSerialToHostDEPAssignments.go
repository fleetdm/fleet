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
			ADD COLUMN hardware_serial varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			ADD INDEX idx_hdep_hardware_serial (hardware_serial)
	`)
	if err != nil {
		return errors.Wrap(err, "add host_dep_assignments.hardware_serial column")
	}

	_, err = tx.Exec(`
		UPDATE host_dep_assignments hda
		JOIN hosts h ON h.id = hda.host_id
		SET hda.hardware_serial = h.hardware_serial
		WHERE hda.deleted_at IS NULL
	`)
	if err != nil {
		return errors.Wrap(err, "backfill host_dep_assignments.hardware_serial column")
	}

	return nil
}

func Down_20260401153503(tx *sql.Tx) error {
	return nil
}
