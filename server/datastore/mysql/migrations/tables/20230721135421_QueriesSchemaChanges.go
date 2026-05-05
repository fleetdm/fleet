package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230721135421, Down_20230721135421)
}

func Up_20230721135421(tx *sql.Tx) error {
	// Drop FK constraint based on queries (name) - since the uniqueness constraint on the queries
	// table changed.
	if _, err := tx.Exec(`
		ALTER TABLE scheduled_queries 
			ADD team_id_char CHAR(10) DEFAULT '' NOT NULL,
			DROP FOREIGN KEY scheduled_queries_query_name;
	`); err != nil {
		return errors.Wrap(err, "removing FK on scheduled_queries")
	}

	if _, err := tx.Exec(`
		ALTER TABLE queries
			DROP INDEX idx_query_unique_name,
			DROP INDEX constraint_query_name_unique,

			ADD team_id INT(10) UNSIGNED DEFAULT NULL,
			ADD team_id_char CHAR(10) DEFAULT '' NOT NULL,

			ADD platform VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT '' NOT NULL,
			ADD min_osquery_version VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT '' NOT NULL,

			ADD schedule_interval INT(10) UNSIGNED DEFAULT 0 NOT NULL,
			ADD automations_enabled TINYINT(1) UNSIGNED DEFAULT 0 NOT NULL,
			ADD logging_type VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT 'snapshot' NOT NULL,

			ADD FOREIGN KEY fk_queries_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE,
			ADD UNIQUE INDEX idx_team_id_name_unq (team_id_char, name);
	`); err != nil {
		return errors.Wrap(err, "updating queries schema")
	}

	// Add new FK constraint to make sure all scheduled_queries exists as 'global' queries.
	if _, err := tx.Exec(`
		ALTER TABLE scheduled_queries 
			ADD FOREIGN KEY fk_scheduled_queries_queries (team_id_char, query_name) REFERENCES queries (team_id_char, name) ON DELETE CASCADE ON UPDATE CASCADE;
	`); err != nil {
		return errors.Wrap(err, "adding new FK on scheduled_queries")
	}

	return nil
}

func Down_20230721135421(tx *sql.Tx) error {
	return nil
}
