package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240905200001, Down_20240905200001)
}

func Up_20240905200001(tx *sql.Tx) error {
	//
	// Changes in `policies` and `policy_stats` to support policies for "No team".
	// "No team" here means policies that run on hosts that belong to no team (hosts.team_id = NULL)
	//
	// `policies`:
	//	- team_id = NULL means the policy is a "Global policy" (aka "All teams" policy).
	//	- team_id > 0 means the policy is a team policy.
	//	- team_id = 0 means the policy is a "No team" policy.
	//
	// `policy_stats`:
	// 	- For "Global policies":
	//	  - inherited_team_id_char = 'global', inherited_team_id = NULL are the stats for the policy's global domain.
	//	  - inherited_team_id_char = '<TEAM_ID>', inherited_team_id = <TEAM_ID> are the stats of the policy on a specific team domain.
	//	  - inherited_team_id_car = '0', inherited_team_id = 0 are the stats of the policy on the "No team" domain.
	//	- For "Team policies" (for team policies there's always just one row in this table):
	//	  - inherited_team_id_char = 'global', inherited_team_id = NULL are the stats for the team policy.
	//

	// Drop foreign key on policies table to teams to allow for team_id = 0 to represent "No team".
	referencedTables := map[string]struct{}{"teams": {}}
	table := "policies"
	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return err
	}
	if len(constraints) != 1 {
		return errors.New("policies foreign key to teams not found")
	}
	if _, err := tx.Exec(fmt.Sprintf(`
		ALTER TABLE policies
		DROP FOREIGN KEY %s;
	`, constraints[0])); err != nil {
		return fmt.Errorf("failed to drop policies foreign key to teams: %w", err)
	}

	// Allow `inherited_team_id` to be NULL to represent global policy stats on the global domain, and `inherited_team_id = 0`
	// to represent global policy stats on the "No team" domain.
	// Add `inherited_team_id_char` as generated column to add uniqueness constraint to the table for policies on each domain.
	if _, err := tx.Exec(`
		ALTER TABLE policy_stats
		DROP INDEX policy_team_unique,
		MODIFY inherited_team_id INT UNSIGNED NULL,
		ADD COLUMN inherited_team_id_char char(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
			GENERATED ALWAYS AS (IF(inherited_team_id IS NULL, 'global', CONVERT(inherited_team_id, CHAR))),
		ADD UNIQUE KEY (policy_id, inherited_team_id_char);
	`); err != nil {
		return fmt.Errorf("failed to modify inherited_team_id in policy_stats: %w", err)
	}

	// Update inherited_team_id from `0` to `NULL` to allow storing stats for the "No team" domain as `inherited_team_id = 0`.
	if _, err := tx.Exec(`
		UPDATE policy_stats
		SET inherited_team_id = NULL
		WHERE inherited_team_id = 0;
	`); err != nil {
		return fmt.Errorf("failed to update policy_stats: %w", err)
	}

	return nil
}

func Down_20240905200001(tx *sql.Tx) error {
	return nil
}
