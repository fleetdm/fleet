package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230906152143, Down_20230906152143)
}

func Up_20230906152143(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE scripts (
    id                 INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,

    -- team_id NULL is for no team (cannot use 0 with foreign key)
    team_id            INT(10) UNSIGNED NULL,

    -- this field is 0 for global, and the team_id otherwise, and is
    -- used for the unique index/constraint (team_id cannot be used
    -- as it allows NULL).
    global_or_team_id  INT(10) UNSIGNED NOT NULL DEFAULT 0,

    -- the name of the script file that was uploaded (or applied via fleetctl), note
    -- that only the filename part of the path is kept and it must be unique for a
    -- given team/no team.
    name               VARCHAR(255) NOT NULL,

    -- the contents of the script to execute, length is limited by Fleet.
    script_contents    TEXT NOT NULL,

    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    UNIQUE KEY idx_scripts_global_or_team_id_name (global_or_team_id, name),
    FOREIGN KEY fk_scripts_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create scripts table: %w", err)
	}

	// alter the host_script_results table to add FK to scripts, and on scripts
	// delete set it to null so that the host script results entry becomes just
	// like an "anonymous" script execution (a one-off, same as those via fleetctl
	// run-script) and stop showing up in the list of scripts executions in the
	// host's page).
	_, err = tx.Exec(`
ALTER TABLE host_script_results
ADD COLUMN script_id INT(10) UNSIGNED NULL DEFAULT NULL,
ADD CONSTRAINT fk_host_script_results_script_id FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("add script_id to host_script_results: %w", err)
	}

	return nil
}

func Down_20230906152143(tx *sql.Tx) error {
	return nil
}
