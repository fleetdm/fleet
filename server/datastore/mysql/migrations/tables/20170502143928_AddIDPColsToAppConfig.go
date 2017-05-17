package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170502143928, Down_20170502143928)
}

func Up_20170502143928(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `entity_id` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' AFTER `osquery_enroll_secret`, " +
			"ADD COLUMN `issuer_uri` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' AFTER `entity_id`, " +
			"ADD COLUMN `idp_image_url` VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' AFTER `issuer_uri`, " +
			"ADD COLUMN `metadata` TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL AFTER `idp_image_url`, " +
			"ADD COLUMN `metadata_url` VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' AFTER `metadata`, " +
			"ADD COLUMN `idp_name` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' AFTER `metadata_url`, " +
			"ADD COLUMN `enable_sso` TINYINT(1) NOT NULL DEFAULT FALSE AFTER `idp_name`; ",
	)
	return err
}

func Down_20170502143928(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"DROP COLUMN `entity_id`, " +
			"DROP COLUMN `issuer_uri`, " +
			"DROP COLUMN `idp_image_url`, " +
			"DROP COLUMN `metadata`, " +
			"DROP COLUMN `metadata_url`, " +
			"DROP COLUMN `idp_name`, " +
			"DROP COLUMN `enable_sso`;",
	)
	return err
}
