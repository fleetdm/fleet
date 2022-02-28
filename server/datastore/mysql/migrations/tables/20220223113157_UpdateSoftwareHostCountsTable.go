package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220223113157, Down_20220223113157)
}

func Up_20220223113157(tx *sql.Tx) error {
	alterStmt := `ALTER TABLE software_host_counts
    ADD COLUMN team_id INT(10) UNSIGNED NOT NULL DEFAULT 0,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (software_id, team_id),
    ADD INDEX idx_software_host_counts_team_id_hosts_count_software_id (team_id,hosts_count,software_id),
    DROP INDEX idx_software_host_counts_host_count_software_id`
	if _, err := tx.Exec(alterStmt); err != nil {
		return errors.Wrap(err, "alter software_host_counts table")
	}
	return nil
}

func Down_20220223113157(tx *sql.Tx) error {
	return nil
}
