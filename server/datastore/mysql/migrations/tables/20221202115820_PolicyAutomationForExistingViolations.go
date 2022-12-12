package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221202115820, Down_20221202115820)
}

func Up_20221202115820(tx *sql.Tx) error {
	for name, query := range map[string]string{
		"create table": `
			CREATE TABLE automation_iterations (
				policyID INTEGER NOT NULL PRIMARY KEY,
				interation INTEGER NOT NULL
			);
		`,
		"alter table": `
			ALTER TABLE policy_membership ADD COLUMN automation_iteration INTEGER NULL;
		`,
	} {
		if _, err := tx.Exec(query); err != nil {
			return errors.Wrap(err, name)
		}
	}
	return nil
}

func Down_20221202115820(tx *sql.Tx) error {
	return nil
}
