package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260812083512, Down_20260812083512)
}

func Up_20260812083512(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE in_house_apps
		ADD COLUMN install_during_setup TINYINT(1) NOT NULL DEFAULT '0'
	`); err != nil {
		return fmt.Errorf("adding install_during_setup to in_house_apps: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE setup_experience_status_results
		ADD COLUMN in_house_app_id INT UNSIGNED DEFAULT NULL,
		ADD CONSTRAINT fk_setup_experience_status_results_iha_id
			FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE
	`); err != nil {
		return fmt.Errorf("adding in_house_app_id to setup_experience_status_results: %w", err)
	}

	return nil
}

func Down_20260812083512(tx *sql.Tx) error {
	return nil
}
