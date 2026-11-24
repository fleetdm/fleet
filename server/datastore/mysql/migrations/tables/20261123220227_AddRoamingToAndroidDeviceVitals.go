package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20261123220227, Down_20261123220227)
}

func Up_20261123220227(tx *sql.Tx) error {
	// Roaming is reported by the Fleet Android agent, not AMAPI, on its own
	// schedule, so it has its own timestamp; updated_at moves with either writer.
	if columnExists(tx, "host_mdm_android_device_vitals", "roaming") {
		return nil
	}
	_, err := tx.Exec(`
ALTER TABLE host_mdm_android_device_vitals
	ADD COLUMN roaming TINYINT(1) DEFAULT NULL,
	ADD COLUMN roaming_updated_at DATETIME(6) DEFAULT NULL`)
	return errors.Wrap(err, "add roaming columns to host_mdm_android_device_vitals")
}

func Down_20261123220227(tx *sql.Tx) error {
	return nil
}
