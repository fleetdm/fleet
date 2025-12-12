package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220915165116, Down_20220915165116)
}

func Up_20220915165116(tx *sql.Tx) error {
	for _, change := range []struct{ name, sql string }{
		{"delete index", `ALTER TABLE hosts DROP INDEX hosts_search`},
		{"create index", `CREATE FULLTEXT INDEX hosts_search ON hosts(hostname, uuid, computer_name)`},
		{"new table", `
			CREATE TABLE host_display_names (
			    host_id int(10) unsigned NOT NULL,
			    display_name varchar(255) NOT NULL,
			    PRIMARY KEY (host_id),
			    KEY (display_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`},
		{"migrate data", `
			INSERT INTO host_display_names (
				SELECT id host_id, IF(computer_name='', hostname, computer_name) display_name FROM hosts
			)
		`},
	} {
		if _, err := tx.Exec(change.sql); err != nil {
			return errors.Wrapf(err, "upHostDisplayName: %s", change.name)
		}
	}
	return nil
}

func Down_20220915165116(tx *sql.Tx) error {
	return nil
}
