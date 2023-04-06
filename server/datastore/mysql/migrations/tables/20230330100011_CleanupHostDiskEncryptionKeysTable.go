package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230330100011, Down_20230330100011)
}

func Up_20230330100011(tx *sql.Tx) error {
	_, err := tx.Exec(`
	DELETE hdek
		FROM host_disk_encryption_keys hdek
		JOIN host_disks hd ON hdek.host_id = hd.host_id
	WHERE hd.encrypted = 0
	`)
	if err != nil {
		return errors.Wrap(err, "cleanup host disk encryption keys")
	}

	return nil
}

func Down_20230330100011(tx *sql.Tx) error {
	return nil
}
