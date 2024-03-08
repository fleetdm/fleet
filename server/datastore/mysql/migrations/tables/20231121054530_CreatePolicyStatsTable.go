package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231121054530, Down_20231121054530)
}

func Up_20231121054530(tx *sql.Tx) error {
	stmt := `
	CREATE TABLE policy_stats (
		id int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
		policy_id int(10) unsigned NOT NULL,
		-- inherited_team_id is used to indicate the row contains inherited 
		-- global policies counts for a team, otherwise it will be 0.  This allows us 
		-- to use the UNIQUE KEY constraint with this column to avoid duplicate rows
		-- when policies.team_id is null. 
		-- A foreign key constraint is not used here because team 0 is not a valid team
		inherited_team_id int(10) unsigned NOT NULL DEFAULT 0,
		-- cached counts for the policy / team
		passing_host_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,
		failing_host_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,
		
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		
		FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE,
		UNIQUE KEY policy_team_unique (policy_id, inherited_team_id)
);
`

	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to create policy_stats table: %w", err)
	}

	return nil
}

func Down_20231121054530(tx *sql.Tx) error {
	return nil
}
