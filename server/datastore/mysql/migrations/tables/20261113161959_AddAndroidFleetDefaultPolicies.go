package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261113161959, Down_20261113161959)
}

// Up_20261113161959 tracks the per-fleet Android default policies that enrollment tokens point at.
// team_id 0 is Unassigned, so it has no foreign key to teams. Existing zero-touch tokens were created
// against the global default policy, so they keep uses_fleet_policy = 0 until they're replaced.
func Up_20261113161959(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS android_fleet_default_policies (
			team_id INT UNSIGNED NOT NULL,
			checksum BINARY(16) NOT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (team_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating android_fleet_default_policies: %w", err)
	}

	if columnExists(tx, "android_zero_touch_tokens", "uses_fleet_policy") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE android_zero_touch_tokens
		ADD COLUMN uses_fleet_policy TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("adding uses_fleet_policy to android_zero_touch_tokens: %w", err)
	}
	return nil
}

func Down_20261113161959(tx *sql.Tx) error {
	return nil
}
