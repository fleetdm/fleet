package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260324223334, Down_20260324223334)
}

func Up_20260324223334(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(
			`ALTER TABLE query_results ADD COLUMN has_data TINYINT(1) GENERATED ALWAYS AS (data IS NOT NULL) VIRTUAL`,
			"adding has_data virtual column to query_results",
		),
		basicMigrationStep(
			`ALTER TABLE query_results ADD INDEX idx_query_id_has_data_host_id_last_fetched (query_id, has_data, host_id, last_fetched)`,
			"adding idx_query_id_has_data_host_id_last_fetched index to query_results",
		),
	}, tx)
}

func Down_20260324223334(tx *sql.Tx) error {
	return nil
}
