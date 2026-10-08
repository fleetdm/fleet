package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260928151511, Down_20260928151511)
}

func Up_20260928151511(tx *sql.Tx) error {
	for _, table := range []string{
		"mdm_apple_configuration_profiles",
		"mdm_apple_declarations",
		"mdm_windows_configuration_profiles",
		"mdm_android_configuration_profiles",
	} {
		if columnExists(tx, table, "description") {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE %s
			ADD COLUMN description VARCHAR(1023) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
		`, table)); err != nil {
			return fmt.Errorf("adding description to %s: %w", table, err)
		}
	}
	return nil
}

func Down_20260928151511(tx *sql.Tx) error {
	return nil
}
