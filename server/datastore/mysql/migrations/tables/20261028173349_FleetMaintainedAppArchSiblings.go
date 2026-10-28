package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261028173349, Down_20261028173349)
}

func Up_20261028173349(tx *sql.Tx) error {
	// The x64 and ARM64 builds of a Windows Fleet-maintained app can now share a title, often at the
	// same version, so the dedup key needs the app as well. Custom packages get 0 and keep deduping by
	// content hash. One ALTER, so a failure leaves the old key in place for a retry.
	if !columnExists(tx, "software_installers", "dedup_fleet_maintained_app_id") {
		if _, err := tx.Exec(`
			ALTER TABLE software_installers
				ADD COLUMN dedup_fleet_maintained_app_id INT UNSIGNED
					GENERATED ALWAYS AS (COALESCE(fleet_maintained_app_id, 0)) VIRTUAL,
				DROP INDEX idx_software_installers_dedup,
				ADD UNIQUE KEY idx_software_installers_dedup (global_or_team_id, title_id, dedup_fleet_maintained_app_id, dedup_token)
		`); err != nil {
			return fmt.Errorf("adding fleet-maintained app to software_installers dedup key: %w", err)
		}
	}

	// Each build keeps its own version pin, since the builds' versions drift apart.
	if !columnExists(tx, "software_title_team_pins", "fleet_maintained_app_id") {
		if _, err := tx.Exec(`
			ALTER TABLE software_title_team_pins
				ADD COLUMN fleet_maintained_app_id INT UNSIGNED NOT NULL AFTER title_id
		`); err != nil {
			return fmt.Errorf("adding fleet_maintained_app_id to software_title_team_pins: %w", err)
		}
	}

	// Until now a title held one Fleet-maintained app per fleet, so the pin belongs to it. Prefer the
	// active row's app in case an older app's rows were left on the title.
	if _, err := tx.Exec(`
		UPDATE software_title_team_pins p
		JOIN (
			SELECT global_or_team_id, title_id,
				COALESCE(MIN(CASE WHEN is_active = 1 THEN fleet_maintained_app_id END), MIN(fleet_maintained_app_id)) AS fleet_maintained_app_id
			FROM software_installers
			WHERE fleet_maintained_app_id IS NOT NULL
			GROUP BY global_or_team_id, title_id
		) fma ON fma.global_or_team_id = p.team_id AND fma.title_id = p.title_id
		SET p.fleet_maintained_app_id = fma.fleet_maintained_app_id
		WHERE p.fleet_maintained_app_id = 0
	`); err != nil {
		return fmt.Errorf("backfilling fleet_maintained_app_id on software_title_team_pins: %w", err)
	}

	var pkHasApp bool
	if err := tx.QueryRow(`
		SELECT COUNT(*) > 0
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'software_title_team_pins'
			AND CONSTRAINT_NAME = 'PRIMARY' AND COLUMN_NAME = 'fleet_maintained_app_id'
	`).Scan(&pkHasApp); err != nil {
		return fmt.Errorf("checking software_title_team_pins primary key: %w", err)
	}
	if !pkHasApp {
		if _, err := tx.Exec(`
			ALTER TABLE software_title_team_pins
				DROP PRIMARY KEY,
				ADD PRIMARY KEY (team_id, title_id, fleet_maintained_app_id)
		`); err != nil {
			return fmt.Errorf("adding fleet_maintained_app_id to software_title_team_pins primary key: %w", err)
		}
	}

	return nil
}

func Down_20261028173349(tx *sql.Tx) error {
	return nil
}
