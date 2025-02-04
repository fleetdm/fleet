package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250127162751, Down_20250127162751)
}

func Up_20250127162751(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE upcoming_activities (
	id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	host_id         INT UNSIGNED NOT NULL,

	-- priority 0 is normal, > 0 is higher priority, < 0 is lower priority.
	priority        INT NOT NULL DEFAULT 0,

	-- user_id is the user that triggered the activity, it may be null if the
	-- activity is fleet-initiated or the user was deleted. Additional user
	-- information (name, email, etc.) is stored in the JSON payload.
	user_id         INT UNSIGNED NULL,
	fleet_initiated TINYINT(1) NOT NULL DEFAULT 0,

	-- type of activity to be executed, currently we only support those, but as
	-- more activity types get added, we can enrich the ENUM with an ALTER TABLE.
	activity_type  ENUM('script', 'software_install', 'software_uninstall', 'vpp_app_install') NOT NULL,

	-- execution_id is the identifier of the activity that will be used when
	-- executed - e.g. scripts and software installs have an execution_id, and
	-- it is sometimes important to know it as soon as the activity is enqueued,
	-- so we need to generate it immediately. Every activity will be identified
	-- via this unique execution_id.
	execution_id   VARCHAR(255) NOT NULL,
	payload        JSON NOT NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	activated_at DATETIME(6) NULL,
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (id),
	UNIQUE KEY idx_upcoming_activities_execution_id (execution_id),
	-- index for the common access pattern to get the next activity to execute
	INDEX idx_upcoming_activities_host_id_priority_created_at (host_id, priority, created_at),
	-- index for the common access pattern to get by activity type (e.g. deleting pending scripts)
	INDEX idx_upcoming_activities_host_id_activity_type (activity_type, host_id),
	CONSTRAINT fk_upcoming_activities_user_id
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`,
	)
	if err != nil {
		return fmt.Errorf("failed to create upcoming_activities: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE script_upcoming_activities (
	upcoming_activity_id       BIGINT UNSIGNED NOT NULL,

	-- those are all columns and not JSON fields because we need FKs on them to
	-- do processing ON DELETE, otherwise we'd have to check for existence of
	-- each one when executing the activity (we need the enqueue next activity
	-- action to be efficient).
	script_id                  INT UNSIGNED NULL,
	script_content_id          INT UNSIGNED NULL,
	policy_id                  INT UNSIGNED NULL,
	setup_experience_script_id INT UNSIGNED NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (upcoming_activity_id),
	CONSTRAINT fk_script_upcoming_activities_upcoming_activity_id
		FOREIGN KEY (upcoming_activity_id) REFERENCES upcoming_activities (id) ON DELETE CASCADE,
	CONSTRAINT fk_script_upcoming_activities_script_id
		FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE SET NULL,
	CONSTRAINT fk_script_upcoming_activities_script_content_id
		FOREIGN KEY (script_content_id) REFERENCES script_contents (id) ON DELETE CASCADE,
	CONSTRAINT fk_script_upcoming_activities_policy_id
		FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE SET NULL,
	CONSTRAINT fk_script_upcoming_activities_setup_experience_script_id
		FOREIGN KEY (setup_experience_script_id) REFERENCES setup_experience_scripts (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE software_install_upcoming_activities (
	upcoming_activity_id       BIGINT UNSIGNED NOT NULL,

	-- those are all columns and not JSON fields because we need FKs on them to
	-- do processing ON DELETE, otherwise we'd have to check for existence of
	-- each one when executing the activity (we need the enqueue next activity
	-- action to be efficient).
	software_installer_id      INT UNSIGNED NULL,
	policy_id                  INT UNSIGNED NULL,
	software_title_id          INT UNSIGNED NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (upcoming_activity_id),
	CONSTRAINT fk_software_install_upcoming_activities_upcoming_activity_id
		FOREIGN KEY (upcoming_activity_id) REFERENCES upcoming_activities (id) ON DELETE CASCADE,
	CONSTRAINT fk_software_install_upcoming_activities_software_installer_id
		FOREIGN KEY (software_installer_id) REFERENCES software_installers (id) ON DELETE SET NULL ON UPDATE CASCADE,
	CONSTRAINT fk_software_install_upcoming_activities_policy_id
		FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE SET NULL,
	CONSTRAINT fk_software_install_upcoming_activities_software_title_id
		FOREIGN KEY (software_title_id) REFERENCES software_titles (id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE vpp_app_upcoming_activities (
	upcoming_activity_id       BIGINT UNSIGNED NOT NULL,

	-- those are all columns and not JSON fields because we need FKs on them to
	-- do processing ON DELETE, otherwise we'd have to check for existence of
	-- each one when executing the activity (we need the enqueue next activity
	-- action to be efficient).
	adam_id                    VARCHAR(16) NOT NULL,
	platform                   VARCHAR(10) NOT NULL,
	vpp_token_id               INT UNSIGNED NULL,
	policy_id                  INT UNSIGNED NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (upcoming_activity_id),
	CONSTRAINT fk_vpp_app_upcoming_activities_upcoming_activity_id
		FOREIGN KEY (upcoming_activity_id) REFERENCES upcoming_activities (id) ON DELETE CASCADE,
	CONSTRAINT fk_vpp_app_upcoming_activities_adam_id_platform
		FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE,
	CONSTRAINT fk_vpp_app_upcoming_activities_vpp_token_id
		FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens (id) ON DELETE SET NULL,
	CONSTRAINT fk_vpp_app_upcoming_activities_policy_id
		FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`,
	)
	if err != nil {
		return err
	}
	return nil
}

func Down_20250127162751(tx *sql.Tx) error {
	return nil
}
