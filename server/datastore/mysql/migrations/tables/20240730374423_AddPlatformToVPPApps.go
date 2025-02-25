package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20240730374423, Down_20240730374423)
}

func Up_20240730374423(tx *sql.Tx) error {
	if columnExists(tx, "vpp_apps", "platform") && columnExists(tx, "vpp_apps_teams", "platform") {
		return nil
	}

	_, err := tx.Exec(`
		ALTER TABLE vpp_apps
		ADD COLUMN platform VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("adding platform to vpp_apps: %w", err)
	}

	_, err = tx.Exec(`UPDATE vpp_apps SET platform = ?, updated_at = updated_at`, fleet.MacOSPlatform)
	if err != nil {
		return fmt.Errorf("updating platform in vpp_apps: %w", err)
	}

	// Drop foreign keys first so they don't interfere with updating primary key.
	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams DROP FOREIGN KEY vpp_apps_teams_ibfk_1`)
	if err != nil {
		return fmt.Errorf("updating foreign key in vpp_apps: %w", err)
	}

	// We drop this foreign key in this migration (for MySQL 8.4). It will be added back in the next migration.
	_, err = tx.Exec(`
		ALTER TABLE host_vpp_software_installs DROP FOREIGN KEY host_vpp_software_installs_ibfk_2`)
	if err != nil {
		return fmt.Errorf("drop foreign key in host_vpp_software_installs: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps DROP PRIMARY KEY, ADD PRIMARY KEY (adam_id, platform)`)
	if err != nil {
		return fmt.Errorf("updating primary key in vpp_apps: %w", err)
	}

	_, err = tx.Exec(`
		ALTER TABLE vpp_apps_teams
		ADD COLUMN platform VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("adding platform to vpp_apps_teams: %w", err)
	}

	_, err = tx.Exec(`UPDATE vpp_apps_teams SET platform = ?`, fleet.MacOSPlatform)
	if err != nil {
		return fmt.Errorf("updating platform in vpp_apps_teams: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams DROP INDEX idx_global_or_team_id_adam_id`)
	if err != nil {
		return fmt.Errorf("dropping unique key in vpp_apps: %w", err)
	}
	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams ADD UNIQUE KEY idx_global_or_team_id_adam_id (global_or_team_id, adam_id, platform)`)
	if err != nil {
		return fmt.Errorf("adding unique key in vpp_apps: %w", err)
	}
	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams DROP INDEX adam_id, ADD INDEX (adam_id, platform)`)
	if err != nil {
		return fmt.Errorf("updating key in vpp_apps: %w", err)
	}
	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams ADD FOREIGN KEY vpp_apps_teams_ibfk_3 (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE`)
	if err != nil {
		return fmt.Errorf("updating foreign key in vpp_apps: %w", err)
	}

	return nil
}

func Down_20240730374423(_ *sql.Tx) error {
	return nil
}
