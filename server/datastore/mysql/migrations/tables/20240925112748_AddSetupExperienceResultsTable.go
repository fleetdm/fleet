package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240925112748, Down_20240925112748)
}

func Up_20240925112748(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE setup_experience_status_results (
	id				INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	host_uuid		VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	-- Type can be one of 'bootstrap-package', 'software-install', 'post-install-script'
	type			VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	name			VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	-- Status can be one of 'pending', 'installing', 'installed', 'failed', 'ran', 'running'
	status			VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	execution_id	VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	error 			VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,

	PRIMARY KEY (id)
)
`)
	if err != nil {
		return fmt.Errorf("failed to create setup_experience_status_results table: %w", err)
	}

	return nil
}

func Down_20240925112748(tx *sql.Tx) error {
	return nil
}
