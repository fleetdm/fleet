package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210421112652, Down_20210421112652)
}

func Up_20210421112652(tx *sql.Tx) error {
	// Use bigint for ID here because MySQL's auto increment handling is going to end up generating a lot of
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS software (
		id bigint unsigned PRIMARY KEY AUTO_INCREMENT,
		name varchar(255) NOT NULL,
		version varchar(255) NOT NULL DEFAULT '',
		source varchar(64) NOT NULL,
        UNIQUE KEY idx_name_version (name, version, source)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`); err != nil {
		return errors.Wrap(err, "create table software")
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_software (
		host_id int unsigned NOT NULL REFERENCES hosts(id),
		software_id bigint unsigned NOT NULL REFERENCES software(id),
        PRIMARY KEY (host_id, software_id)
	)`); err != nil {
		return errors.Wrap(err, "create table host_software")
	}

	return nil
}

func Down_20210421112652(tx *sql.Tx) error {
	return nil
}
