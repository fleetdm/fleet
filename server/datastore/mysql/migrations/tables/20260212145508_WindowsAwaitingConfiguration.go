package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260212145508, Down_20260212145508)
}

func Up_20260212145508(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE mdm_windows_awaiting_configuration (
	device_id VARCHAR(255) NOT NULL PRIMARY KEY,
	awaiting_configuration BOOLEAN NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	return err
}

func Down_20260212145508(tx *sql.Tx) error {
	return nil
}
