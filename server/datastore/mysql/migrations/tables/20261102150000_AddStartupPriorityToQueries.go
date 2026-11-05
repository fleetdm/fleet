package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261102150000, Down_20261102150000)
}

func Up_20261102150000(tx *sql.Tx) error {
	for _, table := range []string{"queries", "scheduled_queries"} {
		if columnExists(tx, table, "startup_priority") {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE %s
			ADD COLUMN startup_priority BIGINT UNSIGNED NULL,
			ALGORITHM=INSTANT
		`, table)); err != nil {
			return fmt.Errorf("add startup_priority to %s: %w", table, err)
		}
	}
	return nil
}

func Down_20261102150000(tx *sql.Tx) error {
	return nil
}
