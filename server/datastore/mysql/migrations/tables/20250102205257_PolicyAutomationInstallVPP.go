package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250102205257, Down_20250102205257)
}

func Up_20250102205257(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE policy_vpp_automations (
			policy_id INT UNSIGNED NOT NULL,
			adam_id VARCHAR(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			platform VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			FOREIGN KEY fk_policy_vpp_automations_policy_id (policy_id) REFERENCES policies (id) ON DELETE CASCADE,
			FOREIGN KEY fk_policy_vpp_automations_app (adam_id, platform) REFERENCES vpp_apps (adam_id, platform)
		)
	`); err != nil {
		return fmt.Errorf("failed to add policy_vpp_automations join table: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_vpp_software_installs
		ADD COLUMN policy_id INT UNSIGNED DEFAULT NULL,
		ADD FOREIGN KEY fk_host_vpp_software_installs_policy_id (policy_id) REFERENCES policies (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("failed to add policy_id to host VPP software installs: %w", err)
	}

	return nil
}

func Down_20250102205257(tx *sql.Tx) error {
	return nil
}
