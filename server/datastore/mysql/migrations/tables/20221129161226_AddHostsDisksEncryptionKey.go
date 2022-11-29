package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20221129161226, Down_20221129161226)
}

func Up_20221129161226(tx *sql.Tx) error {
	// for now, all disk information is "global" for the host (i.e. not collected
	// per volume/disk/partition/etc.).	This may change in the future, but for
	// now we keep it simple and store the encryption key as a simple column in
	// this table, just as for disk space information.
	_, err := tx.Exec(`ALTER TABLE host_disks ADD COLUMN encryption_key TEXT NULL`)
	return err
}

func Down_20221129161226(tx *sql.Tx) error {
	return nil
}
