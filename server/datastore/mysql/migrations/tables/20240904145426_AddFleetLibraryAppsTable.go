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
	-- TODO: maybe use the "full_token" field from homebrew's JSON API response
	-- see e.g. https://formulae.brew.sh/api/cask/1password.json
	token            varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	version          varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	platform         varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	installer_url    varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	-- TODO: is this only the base part of the URL? If not, we need to store that filename (more flexible anyway).
	filename         varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	-- hash of the binary downloaded from installer_url, allows us to validate we got the right bytes
	-- before sending to S3 (and we store installers on S3 under that sha256 hash as identifier).
	sha256           varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,

	created_at       timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at       timestamp(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

	-- TODO: maybe we should leverage the script_contents table and use install/uninstall script ids with FKs here.
	-- If so we need to alter the script_contents cleanup job to take those references into account.
	install_script   text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	uninstall_script text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

	-- TODO: if there's a better unique identifier from homebrew, add it to the table and use it.
	-- Idea is that if homebrew's version is updated, this new version should update the app in the
	-- Fleet library, not create a new entry.
	UNIQUE KEY idx_fleet_library_apps_token (token)
)`)
	if err != nil {
		return fmt.Errorf("creating fleet_library_apps table: %w", err)
	}

	// New column fleet_library_app_id in software_installers allows to keep
	// track of which Fleet library app the installer comes from, if any.
	//
	// TODO: AFAIU it is not crucial to have this, as even if a user adds the
	// same installer _name_ in software_installers (without passing by the Fleet
	// Library, so this fleet_library_app_id would be NULL), it shouldn't see the
	// same _name_ from the Fleet library as available. But it's probably good in
	// any case to keep track of this, even if not obviously useful now.
	_, err = tx.Exec(`
ALTER TABLE software_installers
	ADD COLUMN fleet_library_app_id int unsigned DEFAULT NULL,
	ADD FOREIGN KEY fk_software_installers_fleet_library_app_id (fleet_library_app_id)
		REFERENCES fleet_library_apps (id) ON DELETE SET NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to alter software_installers to add fleet_library_app_id: %w", err)
	}
	return nil
}

func Down_20240904145426(tx *sql.Tx) error {
	return nil
}
