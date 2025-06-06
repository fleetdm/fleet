package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221223174807, Down_20221223174807)
}

func Up_20221223174807(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE hosts MODIFY osquery_host_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL;
		ALTER TABLE hosts ADD INDEX idx_hosts_hardware_serial (hardware_serial)
	`)
	if err != nil {
		return errors.Wrapf(err, "altering hosts table")
	}

	return nil
}

func Down_20221223174807(tx *sql.Tx) error {
	return nil
}
