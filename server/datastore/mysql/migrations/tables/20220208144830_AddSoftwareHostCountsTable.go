package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220208144830, Down_20220208144830)
}

func Up_20220208144830(tx *sql.Tx) error {
	softwareHostCountsTable := `
    CREATE TABLE IF NOT EXISTS software_host_counts (
      software_id bigint(20) unsigned NOT NULL,
      hosts_count int(10) unsigned NOT NULL,
      created_at  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

      PRIMARY KEY (software_id),
      INDEX idx_software_host_counts_host_count_software_id (hosts_count, software_id),
      INDEX idx_software_host_counts_updated_at_software_id (updated_at, software_id)
    );
	`
	if _, err := tx.Exec(softwareHostCountsTable); err != nil {
		return errors.Wrap(err, "create software_host_counts table")
	}
	return nil
}

func Down_20220208144830(tx *sql.Tx) error {
	return nil
}
