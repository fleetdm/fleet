package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20221013084907, Down_20221013084907)
}

func Up_20221013084907(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS feature_scenarios(
    scenario_id INT(10) UNSIGNED NOT NULL,
	query TEXT NOT NULL,
    PRIMARY KEY (scenario_id)
);
`)
	if err != nil {
		return fmt.Errorf("failed to feature_scenarios: %w", err)
	}

	return nil
}

func Down_20221013084907(tx *sql.Tx) error {
	return nil
}
