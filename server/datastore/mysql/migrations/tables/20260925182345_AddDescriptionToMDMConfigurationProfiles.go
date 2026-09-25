package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260925182345, Down_20260925182345)
}

// Free text written by the admin, shown next to the profile name. Same shape
// as teams.description. It is deliberately not part of any checksum or token
// so that editing it never re-delivers the profile.
func Up_20260925182345(tx *sql.Tx) error {
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

func Down_20260925182345(tx *sql.Tx) error {
	return nil
}
