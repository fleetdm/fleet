package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20231024174135, Down_20231024174135)
}

func Up_20231024174135(tx *sql.Tx) error {
	referencedTables := map[string]struct{}{"queries": {}}
	table := "query_results"

	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return err
	}

	if len(constraints) == 0 {
		return errors.New("found no constraints in query_results")
	}

	for _, constraint := range constraints {
		_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE query_results DROP FOREIGN KEY %s;`, constraint))
		if err != nil {
			return errors.Wrapf(err, "dropping fk %s", constraint)
		}
	}
	return nil
}

func Down_20231024174135(tx *sql.Tx) error {
	return nil
}
