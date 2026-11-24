package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261118193608, Down_20261118193608)
}

func Up_20261118193608(tx *sql.Tx) error {
	// Each column is guarded on its own so a partially applied migration can be retried. The defaults keep inserts from
	// servers still on the previous version working during a rolling upgrade.
	columns := []struct{ name, definition string }{
		{"host_uuid", "VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''"},
		{"profile_uuid", "VARCHAR(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''"},
		{"ca_name", "VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		if columnExists(tx, "challenges", column.name) {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE challenges ADD COLUMN %s %s", column.name, column.definition)); err != nil {
			return fmt.Errorf("adding %s to challenges: %w", column.name, err)
		}
	}
	return nil
}

func Down_20261118193608(tx *sql.Tx) error {
	return nil
}
