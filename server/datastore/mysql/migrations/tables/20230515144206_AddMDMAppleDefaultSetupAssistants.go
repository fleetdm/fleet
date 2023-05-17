package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230515144206, Down_20230515144206)
}

func Up_20230515144206(tx *sql.Tx) error {
	_, err := tx.Exec(`
-- macos default setup assistant stores at most 1 profile uuid per team/no team.
-- The default setup assistant is used only if the team does not have a (custom)
-- setup assistant defined in mdm_apple_setup_assistants.
CREATE TABLE mdm_apple_default_setup_assistants (
    id      INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    -- team_id NULL is for no team (cannot use 0 with foreign key)
    team_id INT(10) UNSIGNED NULL,
    -- this field is 0 for global, and the team_id otherwise, and is
    -- used for the unique index/constraint (team_id cannot be used
    -- as it allows NULL).
    global_or_team_id INT(10) UNSIGNED NOT NULL DEFAULT 0,
    profile_uuid VARCHAR(255) NOT NULL DEFAULT '',
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY idx_mdm_default_setup_assistant_global_or_team_id (global_or_team_id),
    FOREIGN KEY fk_mdm_default_setup_assistant_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
);
`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_apple_default_setup_assistants table: %w", err)
	}
	return nil
}

func Down_20230515144206(tx *sql.Tx) error {
	return nil
}
