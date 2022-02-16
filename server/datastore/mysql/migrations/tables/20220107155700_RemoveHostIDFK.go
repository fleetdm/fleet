package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220107155700, Down_20220107155700)
}

func Up_20220107155700(tx *sql.Tx) error {
	tables := []string{
		"host_additional",
		"host_users",
		"policy_membership",
	}
	for _, table := range tables {
		if err := removeHostIDFK(tx, table); err != nil {
			return err
		}
	}

	return nil
}

func removeHostIDFK(tx *sql.Tx, table string) error {
	referencedTables := map[string]struct{}{
		"hosts": {},
	}
	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return errors.Wrap(err, "getting references to hosts table")
	}
	for _, ct := range constraints {
		if _, err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP FOREIGN KEY %s;`, table, ct)); err != nil {
			return errors.Wrapf(err, "dropping %s foreign keys: %s", table, ct)
		}
	}
	return nil
}

func Down_20220107155700(tx *sql.Tx) error {
	return nil
}
