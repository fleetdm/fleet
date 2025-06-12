package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230425082126, Down_20230425082126)
}

func Up_20230425082126(tx *sql.Tx) error {
	_, err := tx.Exec(`
-- macos setup assistant stores at most 1 profile per team/no team
CREATE TABLE mdm_apple_setup_assistants (
    id      INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    -- team_id NULL is for no team (cannot use 0 with foreign key)
    team_id INT(10) UNSIGNED NULL,
    -- this field is 0 for global, and the team_id otherwise, and is
    -- used for the unique index/constraint (team_id cannot be used
    -- as it allows NULL).
    global_or_team_id INT(10) UNSIGNED NOT NULL DEFAULT 0,
    name    TEXT NOT NULL,
    profile JSON NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY idx_mdm_setup_assistant_global_or_team_id (global_or_team_id),
    FOREIGN KEY fk_mdm_setup_assistant_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_apple_setup_assistants table: %w", err)
	}
	return nil
}

func Down_20230425082126(tx *sql.Tx) error {
	return nil
}
