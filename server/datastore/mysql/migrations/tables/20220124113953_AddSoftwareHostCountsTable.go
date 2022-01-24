package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220124113953, Down_20220124113953)
}

func Up_20220124113953(tx *sql.Tx) error {
	softwareHostCountsTable := `
		CREATE TABLE IF NOT EXISTS software_host_counts (
      software_id bigint(20) unsigned NOT NULL,
      host_count  int(10) unsigned NOT NULL,
			created_at  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  timestamp NOT NULL NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			PRIMARY KEY (software_id)
		);
	`
	if _, err := tx.Exec(softwareHostCountsTable); err != nil {
		return errors.Wrap(err, "create software_host_counts table")
	}
	return nil
}

func Down_20220124113953(tx *sql.Tx) error {
	return nil
}
