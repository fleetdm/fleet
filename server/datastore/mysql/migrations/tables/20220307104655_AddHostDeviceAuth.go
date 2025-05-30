package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220307104655, Down_20220307104655)
}

func Up_20220307104655(tx *sql.Tx) error {
	hostDeviceAuthTable := `
    CREATE TABLE IF NOT EXISTS host_device_auth (
        host_id int(10) UNSIGNED NOT NULL,
        token VARCHAR(255) NOT NULL,
        PRIMARY KEY (host_id),
        UNIQUE INDEX idx_host_device_auth_token (token)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(hostDeviceAuthTable); err != nil {
		return errors.Wrap(err, "create host_device_auth table")
	}
	return nil
}

func Down_20220307104655(tx *sql.Tx) error {
	return nil
}
