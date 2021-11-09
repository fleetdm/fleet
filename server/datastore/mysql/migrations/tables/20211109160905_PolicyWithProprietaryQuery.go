package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211109160905, Down_20211109160905)
}

func Up_20211109160905(tx *sql.Tx) error {
	// The VIRTUAL COLUMN is used to enforce the uniqueness of name+team_id for policies.
	// Using the NULLable team_id in idx_policies_unique_name won't work because MySQL won't enforce
	// uniqueness on NULL values, e.g. you can still have two rows with (name: "query1",
	// team_id=NULL).
	// TODO(lucas): Check availability of the featute in the supported MySQL implementations/versions.
	if _, err := tx.Exec(`ALTER TABLE policies
		ADD COLUMN name VARCHAR(255) NOT NULL,
		ADD COLUMN query mediumtext NOT NULL,
		ADD COLUMN description mediumtext NOT NULL,
  		ADD COLUMN author_id int(10) unsigned DEFAULT NULL,
		ADD team_id_x int(10) unsigned GENERATED ALWAYS AS (COALESCE(team_id, 0)) VIRTUAL NOT NULL,

  		ADD KEY idx_policies_author_id (author_id),
  		ADD KEY idx_policies_team_id (team_id),
  		ADD CONSTRAINT policies_queries_ibfk_1 FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE SET NULL
	`); err != nil {
		return errors.Wrap(err, "adding new columns to 'policies'")
	}
	// Remove duplicate global and team policy queries (references).
	if _, err := tx.Exec(`
        DELETE p1 FROM policies AS p1, policies AS p2
		WHERE p1.ID < p2.ID
		AND p1.query_id = p2.query_id AND p1.team_id <=> p2.team_id
    `); err != nil {
		return errors.Wrap(err, "removing duplicates from 'policies'")
	}
	// Migrate existing referenced policies to be propietary policy queries.
	if _, err := tx.Exec(`
        UPDATE policies p
        JOIN queries q
        ON p.query_id = q.id
        SET
            p.name = q.name,
            p.query = q.query,
            p.description = q.description,
			p.author_id = q.author_id
    `); err != nil {
		return errors.Wrap(err, "migrating data from 'queries' to 'policies'")
	}
	// We need to add this index after migration (otherwise it cannot be applied due to empty "name"s).
	if _, err := tx.Exec(`ALTER TABLE policies
		ADD UNIQUE KEY idx_policies_unique_name (name, team_id_x)
	`); err != nil {
		return errors.Wrap(err, "adding unique key (name, team_id_x) to 'policies'")
	}
	// Drop index to queries table.
	if _, err := tx.Exec(`ALTER TABLE policies DROP FOREIGN KEY policies_ibfk_1`); err != nil {
		return errors.Wrap(err, "dropping policies_ibfk_1")
	}
	// Drop key and column "query_id".
	if _, err := tx.Exec(`ALTER TABLE policies DROP KEY fk_policies_query_id`); err != nil {
		return errors.Wrap(err, "dropping fk_policies_query_id")
	}
	if _, err := tx.Exec(`ALTER TABLE policies DROP COLUMN query_id`); err != nil {
		return errors.Wrap(err, "dropping query_id column")
	}
	return nil
}

func Down_20211109160905(tx *sql.Tx) error {
	return nil
}
