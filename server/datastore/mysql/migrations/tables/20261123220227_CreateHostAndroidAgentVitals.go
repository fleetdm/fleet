package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20261123220227, Down_20261123220227)
}

func Up_20261123220227(tx *sql.Tx) error {
	// Separate from host_mdm_android_device_vitals, which every AMAPI status
	// report overwrites in full: the agent reports on its own schedule and
	// writes only the vitals it sends. One nullable column per vital.
	// reported_at moves with every report, even one that changes nothing.
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_android_agent_vitals (
	host_uuid   VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	roaming     TINYINT(1) DEFAULT NULL,
	reported_at DATETIME(6) NOT NULL,
	created_at  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return errors.Wrap(err, "create host_android_agent_vitals")
}

func Down_20261123220227(tx *sql.Tx) error {
	return nil
}
