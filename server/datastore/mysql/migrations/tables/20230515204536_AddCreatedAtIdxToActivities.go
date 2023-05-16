package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230515204536, Down_20230515204536)
}

func Up_20230515204536(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"CREATE INDEX activities_created_at_idx ON activities (created_at);",
	); err != nil {
		return err
	}

	return nil
}

func Down_20230515204536(tx *sql.Tx) error {
	return nil
}
