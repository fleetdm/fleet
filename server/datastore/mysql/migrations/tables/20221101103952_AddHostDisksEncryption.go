package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20221101103952, Down_20221101103952)
}

func Up_20221101103952(tx *sql.Tx) error {
	// for now, all disk information is "global" for the host (i.e. not collected
	// per volume/disk/partition/etc.).	This may change in the future, but for
	// now we keep it simple and store the encrypted flag as a simple column
	// in this table, just as for disk space information.
	_, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN encrypted TINYINT(1) NULL`)
	return err
}

func Down_20221101103952(tx *sql.Tx) error {
	return nil
}
