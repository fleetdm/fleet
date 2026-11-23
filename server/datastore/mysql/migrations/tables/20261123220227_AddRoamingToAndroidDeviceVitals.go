package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20261123220227, Down_20261123220227)
}

func Up_20261123220227(tx *sql.Tx) error {
	// Roaming is reported by the Fleet Android agent, not AMAPI, so it has its own
	// timestamp: updated_at moves with every Pub/Sub status report.
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
