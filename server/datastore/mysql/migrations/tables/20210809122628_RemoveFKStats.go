package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210809122628, Down_20210809122628)
}

func Up_20210809122628(tx *sql.Tx) error {
	rows, err := tx.Query(`select distinct CONSTRAINT_NAME from information_schema.KEY_COLUMN_USAGE where TABLE_NAME = 'scheduled_query_stats' and REFERENCED_TABLE_NAME is not NULL`)
	if err != nil {
		return errors.Wrap(err, "getting scheduled_query_stats constraints")
	}
	defer rows.Close()
	var constraints []string
	for rows.Next() {
		var constraint string
		err = rows.Scan(&constraint)
		if err != nil {
			return errors.Wrap(err, "scheduled_query_stats: scanning constraints")
		}
		constraints = append(constraints, constraint)
		err := rows.Err()
		if err != nil {
			return errors.Wrap(err, "scheduled_query_stats: rows err")
		}
	}
	rows.Close()
	sql := `ALTER TABLE scheduled_query_stats DROP FOREIGN KEY %s`
	for _, constraint := range constraints {
		if _, err := tx.Exec(fmt.Sprintf(sql, constraint)); err != nil {
			return errors.Wrap(err, "drop constraint")
		}
	}

	return nil
}

func Down_20210809122628(tx *sql.Tx) error {
	return nil
}
