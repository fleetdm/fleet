package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260226213628, Down_20260226213628)
}

func Up_20260226213628(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(
			`ALTER TABLE policies ADD COLUMN type ENUM('dynamic', 'patch') NOT NULL DEFAULT 'dynamic'`,
			"adding type column to policies table",
		),
		basicMigrationStep(
			`ALTER TABLE policies ADD COLUMN patch_software_title_id INT UNSIGNED DEFAULT NULL`,
			"adding patch_software_title_id column to policies table",
		),
		basicMigrationStep(
			`ALTER TABLE policies ADD CONSTRAINT fk_patch_software_title_id 
				FOREIGN KEY (patch_software_title_id) REFERENCES software_titles(id) ON DELETE CASCADE`,
			"adding patch_software_title_id foreign key to policies table",
		),
		basicMigrationStep(
			`ALTER TABLE policies ADD UNIQUE INDEX idx_team_id_patch_software_title_id (team_id, patch_software_title_id)`,
			"adding (team_id, patch_software_title_id) unique index to policies table",
		),
	}, tx)
}

func Down_20260226213628(tx *sql.Tx) error {
	return nil
}
