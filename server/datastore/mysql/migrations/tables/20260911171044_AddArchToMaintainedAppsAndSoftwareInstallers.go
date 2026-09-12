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
//
// MySQL commits each DDL statement on its own, so every step is guarded to let a
// partially applied run resume.
func Up_20260911171044(tx *sql.Tx) error {
	if !columnExists(tx, "fleet_maintained_apps", "arch") {
		if _, err := tx.Exec(`
			ALTER TABLE fleet_maintained_apps
				ADD COLUMN arch VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
		`); err != nil {
			return fmt.Errorf("adding fleet_maintained_apps.arch: %w", err)
		}
	}

	if !columnExists(tx, "software_installers", "arch") {
		if _, err := tx.Exec(`
			ALTER TABLE software_installers
				ADD COLUMN arch VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
		`); err != nil {
			return fmt.Errorf("adding software_installers.arch: %w", err)
		}
	}

	// The new key gets its own name so a retry can tell the two apart.
	if !indexExistsTx(tx, "software_installers", "idx_software_installers_dedup_arch") {
		dropOld := ""
		if indexExistsTx(tx, "software_installers", "idx_software_installers_dedup") {
			dropOld = "DROP INDEX idx_software_installers_dedup,"
		}
		if _, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE software_installers
				%s
				ADD UNIQUE KEY idx_software_installers_dedup_arch (global_or_team_id, title_id, arch, dedup_token)
		`, dropOld)); err != nil {
			return fmt.Errorf("rebuilding software_installers dedup key with arch: %w", err)
		}
	}

	if !columnExists(tx, "software_title_team_pins", "fleet_maintained_app_id") {
		if _, err := tx.Exec(`
			ALTER TABLE software_title_team_pins
				ADD COLUMN fleet_maintained_app_id INT UNSIGNED NOT NULL DEFAULT 0
		`); err != nil {
			return fmt.Errorf("adding software_title_team_pins.fleet_maintained_app_id: %w", err)
		}
	}

	// A pin belongs to the Fleet-maintained app that was added to its title first (the
	// same rule the title's patch policy follows), chosen by the lowest installer id so
	// the result is deterministic. Pins with no FMA installer behind them are stale and
	// are dropped. Both statements only touch rows not yet backfilled.
	if _, err := tx.Exec(`
		UPDATE software_title_team_pins p
			JOIN (
				SELECT global_or_team_id, title_id, MIN(id) AS first_id
				FROM software_installers
				WHERE fleet_maintained_app_id IS NOT NULL
				GROUP BY global_or_team_id, title_id
			) first ON first.global_or_team_id = p.team_id AND first.title_id = p.title_id
			JOIN software_installers si ON si.id = first.first_id
		SET p.fleet_maintained_app_id = si.fleet_maintained_app_id
		WHERE p.fleet_maintained_app_id = 0
	`); err != nil {
		return fmt.Errorf("backfilling software_title_team_pins.fleet_maintained_app_id: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM software_title_team_pins WHERE fleet_maintained_app_id = 0`); err != nil {
		return fmt.Errorf("deleting orphaned software_title_team_pins: %w", err)
	}

	if !constraintExists(tx, "software_title_team_pins", "fk_pin_fleet_maintained_app") {
		if _, err := tx.Exec(`
			ALTER TABLE software_title_team_pins
				DROP PRIMARY KEY,
				ADD PRIMARY KEY (team_id, title_id, fleet_maintained_app_id),
				ADD CONSTRAINT fk_pin_fleet_maintained_app FOREIGN KEY (fleet_maintained_app_id)
					REFERENCES fleet_maintained_apps(id) ON DELETE CASCADE
		`); err != nil {
			return fmt.Errorf("rekeying software_title_team_pins: %w", err)
		}
	}

	return nil
}

func Down_20260911171044(tx *sql.Tx) error {
	return nil
}
