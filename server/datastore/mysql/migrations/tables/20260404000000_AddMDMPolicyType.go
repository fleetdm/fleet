package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260404000000, Down_20260404000000)
}

func Up_20260404000000(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(
			"ALTER TABLE policies MODIFY COLUMN `type` enum('dynamic','patch','mdm') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'dynamic'",
			"adding mdm type to policies type enum",
		),
		basicMigrationStep(
			"ALTER TABLE policies ADD COLUMN `mdm_check_definition` json DEFAULT NULL",
			"adding mdm_check_definition column to policies table",
		),
	}, tx)
}

func Down_20260404000000(tx *sql.Tx) error {
	return nil
}
