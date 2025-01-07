package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250106162751, Down_20250106162751)
}

func Up_20250106162751(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE upcoming_activities (
	id             INT UNSIGNED NOT NULL AUTO_INCREMENT,
	host_id        INT UNSIGNED NOT NULL,

	-- user_id is the user that triggered the activity, it may be null if the
	-- activity is fleet-initiated or the user was deleted. Additional user
	-- information (name, email, etc.) is stored in the JSON payload.
	user_id        INT UNSIGNED NULL,

	-- type of activity to be executed, currently we only support those, but as
	-- more activity types get added, we can enrich the ENUM with an ALTER TABLE.
	activity_type  ENUM('script', 'software_install', 'vpp_app_install') NOT NULL,

	-- execution_id is the identifier of the activity that will be used when
	-- executed - e.g. scripts and software installs have an execution_id, and
	-- it is sometimes important to know it as soon as the activity is enqueued,
	-- so we need to generate it immediately.
	execution_id   VARCHAR(255) NOT NULL,

	-- those are all columns and not JSON fields because we need FKs on them to
	-- do processing ON DELETE, otherwise we'd have to check for existence of
	-- each one when executing the activity (we need the enqueue next activity
	-- action to be efficient).
	script_id                  INT UNSIGNED NULL,
	script_content_id          INT UNSIGNED NULL,
	policy_id                  INT UNSIGNED NULL,
	setup_experience_script_id INT UNSIGNED NULL,

	payload                    JSON NOT NULL,

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (id),
	UNIQUE KEY idx_upcoming_activities_execution_id (execution_id),
	INDEX idx_upcoming_activities_host_id_activity_type (host_id, created_at, activity_type),
	CONSTRAINT fk_upcoming_activities_script_id
		FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE SET NULL,
	CONSTRAINT fk_upcoming_activities_script_content_id
		FOREIGN KEY (script_content_id) REFERENCES script_contents (id) ON DELETE CASCADE,
	CONSTRAINT fk_upcoming_activities_policy_id
		FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE SET NULL,
	CONSTRAINT fk_upcoming_activities_setup_experience_script_id
		FOREIGN KEY (setup_experience_script_id) REFERENCES setup_experience_scripts (id) ON DELETE SET NULL,
	CONSTRAINT fk_upcoming_activities_user_id
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`,
	)
	return err
}

func Down_20250106162751(tx *sql.Tx) error {
	return nil
}
