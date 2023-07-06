package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230706141219, Down_20230706141219)
}

func Up_20230706141219(tx *sql.Tx) error {
	// If we want to drop the uniq constraint on queries.name, we first need to remove this
	// constraint on scheduled_queries, due to a FK constraint scheduled_queries (query_name) =>
	// queries (name).
	if _, err := tx.Exec(`
		ALTER TABLE scheduled_queries DROP FOREIGN KEY scheduled_queries_query_name;
	`); err != nil {
		return errors.Wrap(err, "removing FK on scheduled_queries")
	}

	if _, err := tx.Exec(`
		ALTER TABLE queries
			DROP INDEX idx_query_unique_name,
			DROP INDEX constraint_query_name_unique,

			ADD team_id INT(10) UNSIGNED DEFAULT NULL,
			ADD team_id_char CHAR(10) DEFAULT '',

			ADD platform VARCHAR(255) DEFAULT NULL,
			ADD min_osquery_version VARCHAR(255) DEFAULT NULL,

			ADD interval INT(10) UNSIGNED DEFAULT 0,
			ADD automations_enabled TINYINT(1) UNSIGNED DEFAULT 0,
			ADD logging_type VARCHAR(255) DEFAULT 'snapshot',

			ADD FOREIGN KEY fk_queries_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE,
			ADD UNIQUE INDEX idx_team_id_name_unq (team_id_char, name);
	`); err != nil {
		return errors.Wrap(err, "updating queries schema")
	}

	return nil
}

func Down_20230706141219(tx *sql.Tx) error {
	return nil
}
