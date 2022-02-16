package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211116184030, Down_20211116184030)
}

func Up_20211116184030(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE policies
		ADD COLUMN name VARCHAR(255) NOT NULL,
		ADD COLUMN query mediumtext NOT NULL,
		ADD COLUMN description mediumtext NOT NULL,
  		ADD COLUMN author_id int(10) unsigned DEFAULT NULL,
		ADD COLUMN platforms VARCHAR(255) NOT NULL DEFAULT '',

  		ADD KEY idx_policies_author_id (author_id),
  		ADD KEY idx_policies_team_id (team_id),
  		ADD CONSTRAINT policies_queries_ibfk_1 FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE SET NULL
	`); err != nil {
		return errors.Wrap(err, "adding new columns to 'policies'")
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

	// Legacy policy functionality allowed creating two policies with the same
	// referenced query in "queries" (same query_id). The following will
	// rename such conflicting policies by appending a " (id)" suffix.
	var queryIDs []uint
	txx := sqlx.Tx{Tx: tx}
	if err := txx.Select(&queryIDs, `
		SELECT query_id FROM policies
		GROUP BY query_id
		HAVING COUNT(*) > 1
	`); err != nil {
		return errors.Wrap(err, "getting duplicates from 'policies'")
	}
	if len(queryIDs) == 0 {
		// append 0 to avoid empty args error for `sqlx.In`
		queryIDs = append(queryIDs, 0)
	}
	query, args, err := sqlx.In(`
        UPDATE policies
		SET name = CONCAT(name, " (", CONVERT(id, CHAR) ,")")
		WHERE query_id IN (?)`,
		queryIDs,
	)
	if err != nil {
		return errors.Wrap(err, "error building query to rename duplicates from 'policies'")
	}
	if _, err := txx.Exec(query, args...); err != nil {
		return errors.Wrap(err, "renaming duplicates from 'policies'")
	}

	// We need to add the unique key after the population of the name field (otherwise
	// the creation of the unique key fails because of the empty names).
	if _, err := tx.Exec(
		`ALTER TABLE policies ADD UNIQUE KEY idx_policies_unique_name (name);`,
	); err != nil {
		return errors.Wrap(err, "adding idx_policies_unique_name")
	}

	// Removing foreign key to the "queries" table.
	table := "policies"
	referencedTables := map[string]struct{}{"queries": {}}
	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return errors.Wrap(err, "getting references to queries table")
	}
	for _, constraint := range constraints {
		_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE policies DROP FOREIGN KEY %s;`, constraint))
		if err != nil {
			return errors.Wrapf(err, "dropping fk %s", constraint)
		}
	}

	// Drop index and column "query_id".
	indexName, err := indexNameByColumnName(tx, "policies", "query_id")
	if err != nil {
		return errors.Wrap(err, "getting index name to query_id")
	}
	if _, err := tx.Exec(`ALTER TABLE policies DROP KEY ` + indexName); err != nil {
		return errors.Wrap(err, "dropping query_id index")
	}
	if _, err := tx.Exec(`ALTER TABLE policies DROP COLUMN query_id`); err != nil {
		return errors.Wrap(err, "dropping query_id column")
	}
	return nil
}

func Down_20211116184030(tx *sql.Tx) error {
	return nil
}

func indexNameByColumnName(tx *sql.Tx, table, column string) (string, error) {
	const query = `SELECT INDEX_NAME FROM INFORMATION_SCHEMA.STATISTICS 
		WHERE TABLE_NAME = ? AND COLUMN_NAME = ? AND TABLE_SCHEMA = DATABASE();`
	row := tx.QueryRow(query, table, column)
	var indexName string
	err := row.Scan(&indexName)
	if err != nil {
		return "", errors.Wrapf(err, "scanning for index: %s:%s", table, column)
	}
	return indexName, nil
}
