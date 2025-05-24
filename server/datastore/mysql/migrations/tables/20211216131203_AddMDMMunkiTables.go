package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211216131203, Down_20211216131203)
}

func Up_20211216131203(tx *sql.Tx) error {
	mdmTable := `
		CREATE TABLE IF NOT EXISTS host_mdm (
			host_id int(10) UNSIGNED NOT NULL,
			enrolled bool DEFAULT FALSE NOT NULL,
			server_url VARCHAR(255) DEFAULT '' NOT NULL,
			installed_from_dep bool DEFAULT FALSE NOT NULL,
			PRIMARY KEY (host_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(mdmTable); err != nil {
		return errors.Wrap(err, "create host_mdm table")
	}
	munkiInfoTable := `
		CREATE TABLE IF NOT EXISTS host_munki_info (
			host_id int(10) UNSIGNED NOT NULL,
			version VARCHAR(255) DEFAULT '' NOT NULL,
			PRIMARY KEY (host_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(munkiInfoTable); err != nil {
		return errors.Wrap(err, "create host_munki_info table")
	}

	return nil
}

func Down_20211216131203(tx *sql.Tx) error {
	return nil
}
