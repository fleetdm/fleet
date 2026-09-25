package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260925170904, Down_20260925170904)
}

func Up_20260925170904(tx *sql.Tx) error {
	// script_id is already indexed, but the optimizer declines it once one
	// script owns a large share of the table. exit_code makes the lookup selective.
	// InnoDB drops the auto-created fk_host_script_results_script_id, which this
	// index supersedes as the foreign key's backing index.
	return addIndexesTx(tx, "host_script_results",
		indexDef{"idx_host_script_results_script_exit", "script_id, exit_code"})
}

func Down_20260925170904(tx *sql.Tx) error {
	return nil
}
