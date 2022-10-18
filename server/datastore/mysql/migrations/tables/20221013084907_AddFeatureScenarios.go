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
    digest CHAR(40) NOT NULL,
	scenario TEXT NOT NULL,
	n_params INT NOT NULL,
    PRIMARY KEY (digest),
	KEY idx_n_params (n_params)
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
