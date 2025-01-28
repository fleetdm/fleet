package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250124194347, Down_20250124194347)
}

func Up_20250124194347(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		MODIFY COLUMN source VARCHAR(64) CHARACTER SET ascii COLLATE ascii_general_ci;
	`); err != nil {
		return fmt.Errorf("failed to modify source column: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		MODIFY COLUMN bundle_identifier VARCHAR(255) CHARACTER SET ascii COLLATE ascii_general_ci;
	`); err != nil {
		return fmt.Errorf("failed to modify source column: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		DROP INDEX idx_sw_titles,
		ADD UNIQUE KEY idx_sw_titles (name, source, browser, bundle_identifier);
	`); err != nil {
		return fmt.Errorf("failed to add vpp_apps_teams_id to policies: %w", err)
	}

	return nil
}

func Down_20250124194347(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		MODIFY COLUMN source VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
	`); err != nil {
		return fmt.Errorf("failed to modify source column: %w", err)
	}
	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		MODIFY COLUMN bundle_identifier VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
	`); err != nil {
		return fmt.Errorf("failed to modify source column: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_titles
		DROP INDEX idx_sw_titles,
		ADD UNIQUE KEY idx_sw_titles (name, source, browser);
	`); err != nil {
		return fmt.Errorf("failed to add vpp_apps_teams_id to policies: %w", err)
	}

	return nil
}
