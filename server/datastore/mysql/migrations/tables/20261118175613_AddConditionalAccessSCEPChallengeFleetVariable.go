package tables

import (
	"database/sql"
	"fmt"
	"time"
)

func init() {
	MigrationClient.AddMigration(Up_20261118175613, Down_20261118175613)
}

func Up_20261118175613(tx *sql.Tx) error {
	// A fixed created_at keeps schema.sql stable across dumps.
	if _, err := tx.Exec(`INSERT IGNORE INTO fleet_variables (name, is_prefix, created_at)
		VALUES ('FLEET_VAR_CONDITIONAL_ACCESS_SCEP_CHALLENGE', 0, ?)`, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)); err != nil {
		return fmt.Errorf("inserting fleet variable CONDITIONAL_ACCESS_SCEP_CHALLENGE: %w", err)
	}
	return nil
}

func Down_20261118175613(tx *sql.Tx) error {
	return nil
}
