package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210623133615, Down_20210623133615)
}

func Up_20210623133615(tx *sql.Tx) error {
	sql := `
		ALTER TABLE hosts
		CHANGE COLUMN host_name hostname varchar(255) NOT NULL DEFAULT '',
		CHANGE COLUMN physical_memory memory bigint(20) NOT NULL DEFAULT '0',
		CHANGE COLUMN detail_update_time detail_updated_at timestamp NULL DEFAULT NULL,
		CHANGE COLUMN label_update_time label_updated_at timestamp NOT NULL DEFAULT '2000-01-01 00:00:00',
		CHANGE COLUMN last_enroll_time last_enrolled_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "rename columns")
	}

	return nil
}

func Down_20210623133615(tx *sql.Tx) error {
	return nil
}
