package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210818151827, Down_20210818151827)
}

func Up_20210818151827(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT DISTINCT CONSTRAINT_NAME, REFERENCED_TABLE_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_NAME = 'scheduled_query_stats' AND CONSTRAINT_NAME <> 'PRIMARY'`)
	if err != nil {
		return errors.Wrap(err, "getting fk for scheduled_query_stats")
	}
	var constraints []string
	for rows.Next() {
		var constraintName string
		var referencedTable string
		err := rows.Scan(&constraintName, &referencedTable)
		if err != nil {
			return errors.Wrap(err, "scanning fk for scheduled_query_stats")
		}
		if referencedTable == "hosts" || referencedTable == "scheduled_queries" {
			constraints = append(constraints, constraintName)
		}
	}
	if len(constraints) == 0 {
		return errors.New("Found no constraints in scheduled_query_stats")
	}

	for _, constraint := range constraints {
		_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE scheduled_query_stats DROP FOREIGN KEY %s;`, constraint))
		if err != nil {
			return errors.Wrapf(err, "dropping fk %s", constraint)
		}
	}
	return nil
}

func Down_20210818151827(tx *sql.Tx) error {
	return nil
}
