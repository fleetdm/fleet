package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230920091442, Down_20230920091442)
}

func Up_20230920091442(tx *sql.Tx) error {
	stmt := `
	CREATE TABLE policy_stats (
		id int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
		policy_id int(10) unsigned NOT NULL,

		-- team_id used to indicate the row contains counts for inherited 
		-- global policies for a team otherwise it will be NULL.
		-- For every global policy there will be a row for global counts (team_id = NULL)
		-- and a row for each team (team_id = team.id)
		inherited_team_id int(10) unsigned NULL,

		-- cached counts for the policy / team
		passing_host_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,
		failing_host_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,
		
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		
		FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE,
		FOREIGN KEY (inherited_team_id) REFERENCES teams(id) ON DELETE CASCADE,
		UNIQUE KEY policy_team_unique (policy_id, inherited_team_id)
);
`

	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to create policy_stats table: %w", err)
	}

	return nil
}

func Down_20230920091442(tx *sql.Tx) error {
	return nil
}
