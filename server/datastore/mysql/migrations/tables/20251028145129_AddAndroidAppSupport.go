package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251028145129, Down_20251028145129)
}

func Up_20251028145129(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE vpp_apps_teams DROP FOREIGN KEY fk_vpp_apps_teams_vpp_token_id`)
	if err != nil {
		return fmt.Errorf("failed to drop fk from vpp_apps_table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams MODIFY vpp_token_id INT UNSIGNED`)
	if err != nil {
		return fmt.Errorf("failed to make vpp_apps_table.vpp_token_id nullable: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_app_upcoming_activities DROP CONSTRAINT fk_vpp_app_upcoming_activities_adam_id_platform`)
	if err != nil {
		return fmt.Errorf("failed to drop vpp_app_upcoming_activities.adam_id fk: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_vpp_software_installs DROP CONSTRAINT host_vpp_software_installs_ibfk_3`)
	if err != nil {
		return fmt.Errorf("failed to drop host_vpp_software_installs.adam_id fk: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams DROP CONSTRAINT vpp_apps_teams_ibfk_3`)
	if err != nil {
		return fmt.Errorf("failed to drop vpp_apps_teams.adam_id fk: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps MODIFY COLUMN adam_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to increase size of vpp_apps.adam_id: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams MODIFY COLUMN adam_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to increase size of vpp_apps_teams.adam_id: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_app_upcoming_activities ADD CONSTRAINT fk_vpp_app_upcoming_activities_adam_id_platform FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform)`)
	if err != nil {
		return fmt.Errorf("failed to add vpp_app_upcoming_activities.adam_id fk: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_vpp_software_installs ADD CONSTRAINT host_vpp_software_installs_ibfk_3 FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to add host_vpp_software_installs.adam_id fk: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_apps_teams ADD CONSTRAINT vpp_apps_teams_ibfk_3 FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to add vpp_apps_teams.adam_id fk: %w", err)
	}

	return nil
}

func Down_20251028145129(tx *sql.Tx) error {
	return nil
}
