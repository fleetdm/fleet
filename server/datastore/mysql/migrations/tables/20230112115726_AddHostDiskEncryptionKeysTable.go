package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230112115726, Down_20230112115726)
}

func Up_20230112115726(tx *sql.Tx) error {
	_, err := tx.Exec(`
  CREATE TABLE host_disk_encryption_keys (
    host_id              INT(10) UNSIGNED NOT NULL,
    disk_encryption_key  BLOB NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (host_id)
  )`)
	if err != nil {
		return errors.Wrapf(err, "create host_disk_encryption_keys table")
	}

	return nil
}

func Down_20230112115726(tx *sql.Tx) error {
	return nil
}
