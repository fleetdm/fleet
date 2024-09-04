package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240904145426, Down_20240904145426)
}

func Up_20240904145426(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE fleet_library_apps (
	id               int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
	name             varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	version          varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	platform         varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	installer_url    varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	-- TODO: is this only the base part of the URL? If not, we need to store that filename (more flexible anyway).
	filename         varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,

	created_at       timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at       timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

	-- TODO: maybe we should leverage the script_contents table and use install/uninstall script ids with FKs here.
	-- If so we need to alter the script_contents cleanup job to take those references into account.
	install_script   text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	uninstall_script text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

	-- TODO: if there's a better unique identifier from homebrew, add it to the table and use it.
	-- Idea is that if homebrew's version is updated, this new version should update the app in the 
	-- Fleet library, not create a new entry.
	UNIQUE KEY idx_fleet_library_apps_name (name)
)`)
	if err != nil {
		return fmt.Errorf("creating fleet_library_apps table: %w", err)
	}
	return nil
}

func Down_20240904145426(tx *sql.Tx) error {
	return nil
}
