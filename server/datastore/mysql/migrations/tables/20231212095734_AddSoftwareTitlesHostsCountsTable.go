package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20231212095734, Down_20231212095734)
}

func Up_20231212095734(tx *sql.Tx) error {
	softwareTitlesHostCountsTable := `
    CREATE TABLE IF NOT EXISTS software_titles_host_counts (
      software_title_id  int(10) unsigned NOT NULL,
      hosts_count        int(10) unsigned NOT NULL,
      team_id            int(10) unsigned NOT NULL DEFAULT 0,
      created_at         timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at         timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

      PRIMARY KEY (software_title_id, team_id),
      INDEX idx_software_titles_host_counts_team_counts_title (team_id,hosts_count,software_title_id),
      INDEX idx_software_titles_host_counts_updated_at_software_title_id (updated_at, software_title_id)
    );
	`
	if _, err := tx.Exec(softwareTitlesHostCountsTable); err != nil {
		return errors.Wrap(err, "create software_titles_host_counts table")
	}
	return nil
}

func Down_20231212095734(tx *sql.Tx) error {
	return nil
}
