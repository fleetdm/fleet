package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260911171044, Down_20260911171044)
}

// Up_20260911171044 lets a Windows software title hold one Fleet-maintained app per
// installer architecture (x64 and ARM64 builds of the same app). The catalog and
// installers carry the architecture, the installer dedup key includes it, and version
// pins are scoped to the Fleet-maintained app instead of the title.
func Up_20260911171044(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE fleet_maintained_apps
			ADD COLUMN arch VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
	`); err != nil {
		return fmt.Errorf("adding fleet_maintained_apps.arch: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_installers
			ADD COLUMN arch VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			DROP INDEX idx_software_installers_dedup,
			ADD UNIQUE KEY idx_software_installers_dedup (global_or_team_id, title_id, arch, dedup_token)
	`); err != nil {
		return fmt.Errorf("adding software_installers.arch and rebuilding dedup key: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_title_team_pins
			ADD COLUMN fleet_maintained_app_id INT UNSIGNED NOT NULL DEFAULT 0
	`); err != nil {
		return fmt.Errorf("adding software_title_team_pins.fleet_maintained_app_id: %w", err)
	}

	// A pin belongs to the Fleet-maintained app whose installer is active on the
	// title. Pins with no such installer are stale and are dropped.
	if _, err := tx.Exec(`
		UPDATE software_title_team_pins p
			JOIN software_installers si
				ON si.global_or_team_id = p.team_id AND si.title_id = p.title_id
				AND si.fleet_maintained_app_id IS NOT NULL AND si.is_active = 1
		SET p.fleet_maintained_app_id = si.fleet_maintained_app_id
	`); err != nil {
		return fmt.Errorf("backfilling software_title_team_pins.fleet_maintained_app_id: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM software_title_team_pins WHERE fleet_maintained_app_id = 0`); err != nil {
		return fmt.Errorf("deleting orphaned software_title_team_pins: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_title_team_pins
			DROP PRIMARY KEY,
			ADD PRIMARY KEY (team_id, title_id, fleet_maintained_app_id),
			ADD CONSTRAINT fk_pin_fleet_maintained_app FOREIGN KEY (fleet_maintained_app_id)
				REFERENCES fleet_maintained_apps(id) ON DELETE CASCADE
	`); err != nil {
		return fmt.Errorf("rekeying software_title_team_pins: %w", err)
	}

	return nil
}

func Down_20260911171044(tx *sql.Tx) error {
	return nil
}
