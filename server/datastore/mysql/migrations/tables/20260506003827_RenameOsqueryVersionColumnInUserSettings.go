package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260506003827, Down_20260506003827)
}

func Up_20260506003827(tx *sql.Tx) error {
	// Rename "osquery_version" to "agent" in users' hidden_host_columns settings.
	// This preserves existing users' column visibility preferences when the
	// Osquery column is replaced by the new Agent column.
	if _, err := tx.Exec(`
		UPDATE users
		SET settings = JSON_SET(
			settings,
			'$.hidden_host_columns',
			CAST(
				REPLACE(
					JSON_EXTRACT(settings, '$.hidden_host_columns'),
					'"osquery_version"',
					'"agent"'
				) AS JSON
			)
		)
		WHERE JSON_CONTAINS(settings, '"osquery_version"', '$.hidden_host_columns')
	`); err != nil {
		return fmt.Errorf("failed to rename osquery_version to agent in user settings: %w", err)
	}
	return nil
}

func Down_20260506003827(tx *sql.Tx) error {
	return nil
}
