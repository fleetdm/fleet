package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221216115821, Down_20221216115821)
}

func Up_20221216115821(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE hosts MODIFY osquery_host_id VARCHAR(255) NULL;
		ALTER TABLE hosts ADD INDEX idx_hosts_hardware_serial (hardware_serial)
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20221216115821(tx *sql.Tx) error {
	return nil
}
