package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20170111133013, Down_20170111133013)
}

func Up_20170111133013(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `queries` " +
			"ADD CONSTRAINT `constraint_query_name_unique` " +
			"UNIQUE (`name`);",
	)
	return err
}

func Down_20170111133013(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `queries` " +
			"DROP INDEX `constraint_query_name_unique`;",
	)
	return err
}
