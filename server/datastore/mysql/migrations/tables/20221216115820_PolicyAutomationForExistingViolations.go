package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221216115820, Down_20221216115820)
}

func Up_20221216115820(tx *sql.Tx) error {
	for name, query := range map[string]string{
		"create table": `
			CREATE TABLE policy_automation_iterations (
				policy_id INT UNSIGNED NOT NULL PRIMARY KEY,
				iteration INT NOT NULL,
				FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE
			);
		`,
		"alter table": `
			ALTER TABLE policy_membership ADD COLUMN automation_iteration INT NULL;
		`,
	} {
		if _, err := tx.Exec(query); err != nil {
			return errors.Wrap(err, name)
		}
	}
	return nil
}

func Down_20221216115820(tx *sql.Tx) error {
	return nil
}
