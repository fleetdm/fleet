package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251007145400, Down_20251007145400)
}

func Up_20251007145400(tx *sql.Tx) error {
	// Note that at the moment of this migration, in-house apps uninstall is not
	// supported, so we don't add it to the enum.
	_, err := tx.Exec(`
ALTER TABLE upcoming_activities
	CHANGE COLUMN activity_type activity_type ENUM('script', 'software_install', 'software_uninstall', 'vpp_app_install', 'in_house_app_install')
		COLLATE utf8mb4_unicode_ci NOT NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter upcoming_activities activity_type: %w", err)
	}

	// Note that at the moment of this migration, auto-install and self-service is not
	// supported for in-house apps, so we don't need to add columns for e.g. policy_id.
	// See https://www.figma.com/design/zcc45sBgdiDZT11iKjLolh/-30936-Deploy-custom--in-house--iOS-app?node-id=5363-11227&t=S1pEnokvQ83v8eJk-0
	_, err = tx.Exec(`
CREATE TABLE in_house_app_upcoming_activities (
	upcoming_activity_id       BIGINT UNSIGNED NOT NULL,

	-- those are all columns and not JSON fields because we need FKs on them to
	-- do processing ON DELETE, otherwise we'd have to check for existence of
	-- each one when executing the activity (we need the enqueue next activity
	-- action to be efficient).
	in_house_app_id            INT UNSIGNED NOT NULL,

	software_title_id          INT UNSIGNED DEFAULT NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (upcoming_activity_id),
	CONSTRAINT fk_in_house_app_upcoming_activities_upcoming_activity_id
		FOREIGN KEY (upcoming_activity_id) REFERENCES upcoming_activities (id) ON DELETE CASCADE,
	CONSTRAINT fk_in_house_app_upcoming_activities_in_house_app_id
		FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE,
	CONSTRAINT fk_in_house_app_upcoming_activities_software_title_id
		FOREIGN KEY (software_title_id) REFERENCES software_titles (id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`,
	)
	if err != nil {
		return fmt.Errorf("failed to create in_house_app_upcoming_activities table: %w", err)
	}

	// Note that at the time of this migration, in-house apps do not support
	// auto-install and self-service installs so those columns have not been added.
	// See https://www.figma.com/design/zcc45sBgdiDZT11iKjLolh/-30936-Deploy-custom--in-house--iOS-app?node-id=5363-11227&t=S1pEnokvQ83v8eJk-0
	_, err = tx.Exec(`
-- This table is the in-house app equivalent of the host_vpp_software_installs table.
-- It tracks the installation of in-house software on particular hosts.
CREATE TABLE host_in_house_software_installs (
	id              INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	host_id         INT(10) UNSIGNED NOT NULL,

	in_house_app_id INT(10) UNSIGNED NOT NULL,

	-- This is the UUID of the MDM command issued to install the app
	command_uuid    VARCHAR(127) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	user_id         INT(10) UNSIGNED NULL,
	platform        VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	removed         TINYINT NOT NULL DEFAULT '0',
	canceled        TINYINT NOT NULL DEFAULT '0',

	verification_command_uuid VARCHAR(127) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
	verification_at           DATETIME(6) DEFAULT NULL,
	verification_failed_at    DATETIME(6) DEFAULT NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at      DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at      DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY(id),
	UNIQUE INDEX idx_host_in_house_software_installs_command_uuid (command_uuid),
	CONSTRAINT fk_host_in_house_software_installs_user_id
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL,
	CONSTRAINT fk_host_in_house_software_installs_in_house_app_id
		FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE,
	INDEX idx_host_in_house_software_installs_verification ((verification_at IS NULL AND verification_failed_at IS NULL))
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table host_in_house_software_installs: %w", err)
	}

	return nil
}

func Down_20251007145400(tx *sql.Tx) error {
	return nil
}
