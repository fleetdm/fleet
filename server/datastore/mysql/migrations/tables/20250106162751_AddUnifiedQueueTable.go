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

	-- type of activity to be executed, currently we only support those, but as
	-- more activity types get added, we can enrich the ENUM with an ALTER TABLE.
	activity_type  ENUM('script', 'software_install', 'vpp_app_install') NOT NULL,

	-- execution_id is the identifier of the activity that will be used when 
	-- executed - e.g. scripts and software installs have an execution_id, and 
	-- it is sometimes important to know it as soon as the activity is enqueued,
	-- so we need to generate it immediately.
	execution_id   VARCHAR(255) NOT NULL,

	value BLOB NOT NULL, -- 64KB max value size

	-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues
	created_at   DATETIME(6) NOT NULL DEFAULT NOW(6),
	updated_at   DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),

	PRIMARY KEY (id),
	CONSTRAINT idx_secret_variables_name UNIQUE (name)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`,
	)
	return err
}

func Down_20250106162751(tx *sql.Tx) error {
	return nil
}
