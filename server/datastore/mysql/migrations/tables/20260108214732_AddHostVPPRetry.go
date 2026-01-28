package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260108214732, Down_20260108214732)
}

func Up_20260108214732(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE host_vpp_software_installs
	ADD COLUMN retry_count INT NOT NULL DEFAULT 0
	`)
	if err != nil {
		return errors.Wrap(err, "add retry_count column to host_vpp_software_installs")
	}
	return nil
}

func Down_20260108214732(tx *sql.Tx) error {
	return nil
}
